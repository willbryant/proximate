package response_cache

import "io"
import "sync"

type File interface {
	io.Reader
	io.ReaderAt
	io.Writer
	io.Closer
	Sync() error;
}

type SharedFile struct {
	cond   sync.Cond
	file   File
	refs   int64
	length int64
	err    error
}

func NewSharedFile(f File) *SharedFile {
	return &SharedFile{
		cond:   sync.Cond{L: &sync.Mutex{}},
		file:   f,
		refs:   1,
		length: 0,
	}
}

func (sf *SharedFile) unreference() (err error) {
	sf.refs -= 1
	if sf.refs == 0 {
		err := sf.file.Close()
		if err != nil {
			sf.err = err
		}
	}
	return nil
}

func (sf *SharedFile) reference() error {
	if sf.refs == 0 {
		return io.EOF
	}
	sf.refs += 1
	return nil
}

func (sf *SharedFile) Write(p []byte) (n int, err error) {
	n, err = sf.file.Write(p)

	sf.cond.L.Lock()
	defer sf.cond.L.Unlock()

	sf.length += int64(n)
	if err != nil {
		sf.err = err
	}

	sf.cond.Broadcast()
	return n, err
}

func (sf *SharedFile) Sync() (err error) {
	err = sf.file.Sync()

	if err != nil {
		sf.Abort(err)
	}

	return err
}

func (sf *SharedFile) Abort(err error) {
	sf.cond.L.Lock()
	defer sf.cond.L.Unlock()

	sf.unreference()
	sf.err = err

	sf.cond.Broadcast()
}

func (sf *SharedFile) Close() (err error) {
	sf.cond.L.Lock()
	defer sf.cond.L.Unlock()

	err = sf.unreference()
	if sf.err == nil {
		sf.err = io.EOF
	}

	sf.cond.Broadcast()

	return err
}

type blockingReader struct {
	sf       *SharedFile
	position int64
}

func (sf *SharedFile) SpawnReader() (io.ReadCloser, error) {
	sf.cond.L.Lock()
	defer sf.cond.L.Unlock()

	err := sf.reference()
	if err != nil {
		return nil, err
	}

	return &blockingReader{
		sf:       sf,
		position: 0,
	}, nil
}

func (reader *blockingReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	for {
		// try to read without the overhead of acquiring the mutex
		bread, err := reader.sf.file.ReadAt(p, reader.position)
		reader.position += int64(bread)

		// if an IO error (other than EOF) occurs, return it
		if err != nil && err != io.EOF {
			return bread, err
		}

		// otherwise if we managed to read anything, return it; note that even if we hit EOF, it's
		// possible that more will be written to the stream later, so we don't return EOF here
		if bread > 0 {
			return bread, nil
		}

		// nothing left to read in the file (so far), so wait for more to arrive
		err = reader.waitForMore()

		// if that indicates that we've reached the true EOF, or any other error, return that
		if err != nil {
			return 0, err
		}
	}
}

func (reader *blockingReader) waitForMore() error {
	reader.sf.cond.L.Lock()
	defer reader.sf.cond.L.Unlock()

	for {
		if reader.sf.length > reader.position {
			return nil
		}
		if reader.sf.err != nil {
			return reader.sf.err
		}
		reader.sf.cond.Wait()
	}
}

func (reader *blockingReader) Close() error {
	reader.sf.cond.L.Lock()
	defer reader.sf.cond.L.Unlock()

	return reader.sf.unreference()
}
