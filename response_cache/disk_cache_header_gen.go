package response_cache

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *DiskCacheHeader) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zcmr uint32
	zcmr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zcmr > 0 {
		zcmr--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "version":
			z.Version, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "status":
			z.Status, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "header":
			var zajw uint32
			zajw, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Header == nil && zajw > 0 {
				z.Header = make(map[string][]string, zajw)
			} else if len(z.Header) > 0 {
				for key, _ := range z.Header {
					delete(z.Header, key)
				}
			}
			for zajw > 0 {
				zajw--
				var zxvk string
				var zbzg []string
				zxvk, err = dc.ReadString()
				if err != nil {
					return
				}
				var zwht uint32
				zwht, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(zbzg) >= int(zwht) {
					zbzg = (zbzg)[:zwht]
				} else {
					zbzg = make([]string, zwht)
				}
				for zbai := range zbzg {
					zbzg[zbai], err = dc.ReadString()
					if err != nil {
						return
					}
				}
				z.Header[zxvk] = zbzg
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *DiskCacheHeader) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "version"
	err = en.Append(0x83, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Version)
	if err != nil {
		return
	}
	// write "status"
	err = en.Append(0xa6, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Status)
	if err != nil {
		return
	}
	// write "header"
	err = en.Append(0xa6, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Header)))
	if err != nil {
		return
	}
	for zxvk, zbzg := range z.Header {
		err = en.WriteString(zxvk)
		if err != nil {
			return
		}
		err = en.WriteArrayHeader(uint32(len(zbzg)))
		if err != nil {
			return
		}
		for zbai := range zbzg {
			err = en.WriteString(zbzg[zbai])
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *DiskCacheHeader) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "version"
	o = append(o, 0x83, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt(o, z.Version)
	// string "status"
	o = append(o, 0xa6, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	o = msgp.AppendInt(o, z.Status)
	// string "header"
	o = append(o, 0xa6, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72)
	o = msgp.AppendMapHeader(o, uint32(len(z.Header)))
	for zxvk, zbzg := range z.Header {
		o = msgp.AppendString(o, zxvk)
		o = msgp.AppendArrayHeader(o, uint32(len(zbzg)))
		for zbai := range zbzg {
			o = msgp.AppendString(o, zbzg[zbai])
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DiskCacheHeader) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zhct uint32
	zhct, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zhct > 0 {
		zhct--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "version":
			z.Version, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "status":
			z.Status, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "header":
			var zcua uint32
			zcua, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Header == nil && zcua > 0 {
				z.Header = make(map[string][]string, zcua)
			} else if len(z.Header) > 0 {
				for key, _ := range z.Header {
					delete(z.Header, key)
				}
			}
			for zcua > 0 {
				var zxvk string
				var zbzg []string
				zcua--
				zxvk, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				var zxhx uint32
				zxhx, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(zbzg) >= int(zxhx) {
					zbzg = (zbzg)[:zxhx]
				} else {
					zbzg = make([]string, zxhx)
				}
				for zbai := range zbzg {
					zbzg[zbai], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
				z.Header[zxvk] = zbzg
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *DiskCacheHeader) Msgsize() (s int) {
	s = 1 + 8 + msgp.IntSize + 7 + msgp.IntSize + 7 + msgp.MapHeaderSize
	if z.Header != nil {
		for zxvk, zbzg := range z.Header {
			_ = zbzg
			s += msgp.StringPrefixSize + len(zxvk) + msgp.ArrayHeaderSize
			for zbai := range zbzg {
				s += msgp.StringPrefixSize + len(zbzg[zbai])
			}
		}
	}
	return
}
