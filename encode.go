package gohessian

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"time"
	"unicode/utf8"
)

/*
对 基本数据进行 Hessian 编码
支持:
	- int int32 int64
	- float64
	- time.Time
	- []byte
	- []interface{}
	- map[interface{}]interface{}
	- nil
	- bool
	- object // 预计支持中...
*/

type Encoder struct {
}

const (
	CHUNK_SIZE    = 0x8000
	ENCODER_DEBUG = false
)

func init() {
	_, filename, _, _ := runtime.Caller(1)
	if ENCODER_DEBUG {
		log.SetPrefix(filename + "\n")
	}
}

// Encode do encode var to binary under hessian protocol
func Encode(v interface{}) (b []byte, err error) {

	switch v.(type) {

	case []byte:
		b, err = encodeBinary(v.([]byte))

	case bool:
		b, err = encodeBool(v.(bool))

	case time.Time:
		b, err = encodeTime(v.(time.Time))

	case float64:
		b, err = encodeFloat64(v.(float64))

	case int:
		if v.(int) >= -2147483648 && v.(int) <= 2147483647 {
			b, err = encodeInt32(int32(v.(int)))
		} else {
			b, err = encodeInt64(int64(v.(int)))
		}

	case int32:
		b, err = encodeInt32(v.(int32))

	case int64:
		b, err = encodeInt64(v.(int64))

	case string:
		b, err = encodeString(v.(string))

	case nil:
		b, err = encodeNull(v)

	case []Any:
		b, err = encodeList(v.([]Any))

	case map[Any]Any:
		b, err = encodeMap(v.(map[Any]Any))

	case Any:
		b, err = encodeObject(v.(Any))

	default:
		return nil, errors.New("unknow type")
	}
	if ENCODER_DEBUG {
		log.Println(SprintHex(b))
	}
	return
}

//=====================================
//对各种数据类型的编码
//=====================================

// binary
func encodeBinary(v []byte) (b []byte, err error) {
	var (
		tag  byte
		lenB []byte
		lenN int
	)

	if len(v) == 0 {
		if lenB, err = PackUint16(0); err != nil {
			b = nil
			return
		}
		b = append(b, 'B')
		b = append(b, lenB...)
		return
	}

	rBuf := *bytes.NewBuffer(v)

	for rBuf.Len() > 0 {
		if rBuf.Len() > CHUNK_SIZE {
			tag = 'b'
			if lenB, err = PackUint16(uint16(CHUNK_SIZE)); err != nil {
				b = nil
				return
			}
			lenN = CHUNK_SIZE
		} else {
			tag = 'B'
			if lenB, err = PackUint16(uint16(rBuf.Len())); err != nil {
				b = nil
				return
			}
			lenN = rBuf.Len()
		}
		b = append(b, tag)
		b = append(b, lenB...)
		b = append(b, rBuf.Next(lenN)...)
	}
	return
}

// boolean
func encodeBool(v bool) (b []byte, err error) {
	if v == true {
		b = append(b, 'T')
		return
	}
	b = append(b, 'F')
	return
}

// date
func encodeTime(v time.Time) (b []byte, err error) {
	var tmpV []byte
	b = append(b, 'd')
	if tmpV, err = PackInt64(v.UnixNano() / 1000000); err != nil {
		b = nil
		return
	}
	b = append(b, tmpV...)
	return
}

// double
func encodeFloat64(v float64) (b []byte, err error) {
	var tmpV []byte
	if tmpV, err = PackFloat64(v); err != nil {
		b = nil
		return
	}
	b = append(b, 'D')
	b = append(b, tmpV...)
	return
}

// int
func encodeInt32(v int32) (b []byte, err error) {
	var tmpV []byte
	if tmpV, err = PackInt32(v); err != nil {
		b = nil
		return
	}
	b = append(b, 'I')
	b = append(b, tmpV...)
	return
}

// long
func encodeInt64(v int64) (b []byte, err error) {
	var tmpV []byte
	if tmpV, err = PackInt64(v); err != nil {
		b = nil
		return
	}
	b = append(b, 'L')
	b = append(b, tmpV...)
	return

}

// null
func encodeNull(v interface{}) (b []byte, err error) {
	b = append(b, 'N')
	return
}

// string
func encodeString(v string) (b []byte, err error) {
	var (
		lenB []byte
		sBuf = *bytes.NewBufferString(v)
		rLen = utf8.RuneCountInString(v)

		sChunk = func(_len int) {
			for i := 0; i < _len; i++ {
				if r, s, err := sBuf.ReadRune(); s > 0 && err == nil {
					b = append(b, []byte(string(r))...)
				}
			}
		}
	)

	if v == "" {
		if lenB, err = PackUint16(uint16(rLen)); err != nil {
			b = nil
			return
		}
		b = append(b, 'S')
		b = append(b, lenB...)
		b = append(b, []byte{}...)
		return
	}

	for {
		rLen = utf8.RuneCount(sBuf.Bytes())
		if rLen == 0 {
			break
		}
		if rLen > CHUNK_SIZE {
			if lenB, err = PackUint16(uint16(CHUNK_SIZE)); err != nil {
				b = nil
				return
			}
			b = append(b, 's')
			b = append(b, lenB...)
			sChunk(CHUNK_SIZE)
		} else {
			if lenB, err = PackUint16(uint16(rLen)); err != nil {
				b = nil
				return
			}
			b = append(b, 'S')
			b = append(b, lenB...)
			sChunk(rLen)
		}
	}
	return
}

// list
func encodeList(v []Any) (b []byte, err error) {
	listLen := len(v)
	var (
		lenB []byte
		tmpV []byte
	)

	b = append(b, 'V')

	if lenB, err = PackInt32(int32(listLen)); err != nil {
		b = nil
		return
	}
	b = append(b, 'l')
	b = append(b, lenB...)

	for _, a := range v {
		if tmpV, err = Encode(a); err != nil {
			b = nil
			return
		}
		b = append(b, tmpV...)
	}
	b = append(b, 'z')
	return
}

// map
func encodeMap(v map[Any]Any) (b []byte, err error) {
	var (
		tmpK []byte
		tmpV []byte
	)
	b = append(b, 'M')
	for k, v := range v {
		if tmpK, err = Encode(k); err != nil {
			b = nil
			return
		}
		if tmpV, err = Encode(v); err != nil {
			b = nil
			return
		}
		b = append(b, tmpK...)
		b = append(b, tmpV...)
	}
	b = append(b, 'z')
	return
}

// object
func encodeObject(v Any) (b []byte, err error) {
	valueV := reflect.ValueOf(v)
	typeV := reflect.TypeOf(v)
	fmt.Println("v => ", v)

	jiji, exist := typeV.FieldByName(ObjectType)
	fmt.Println(jiji)
	fmt.Println(exist)
	if !exist {
		b = nil
		err = errors.New("Object Type not Set")
		return b, err
	}
	objectTypeField := valueV.FieldByName(ObjectType)
	if objectTypeField.Type().String() != "string" {
		b = nil
		err = errors.New("type of Type Field is not String")
		return b, err
	}
	// Object Type
	b = append(b, 'C')
	b = append(b, len(objectTypeField.String()), []byte(objectTypeField.String())...)

	// Object Field Length
	if lenField, err := PackInt16(0x90 + int16(typeV.NumField()) - 1); err != nil { // -1 是为了排除 Type Field
		b = nil
		err = errors.New("can not count field length, error: " + err.Error())
		return b, err
	} else {
		b = append(b, lenField...)
	}

	// Every Field Name
	for i := 0; i < typeV.NumField(); i++ {
		if typeV.Field(i).Name == ObjectType {
			continue
		}
		if name, err := Encode(typeV.Field(i).Name); err != nil {
			b = nil
			err = errors.New("encode field name failed, error: " + err.Error())
			return b, err
		} else {
			b = append(b, name...)
		}
	}

	b = append(b, 'O')
	// Object Value
	for i := 0; i < typeV.NumField(); i++ {
		if typeV.Field(i).Name == ObjectType {
			continue
		}
		typeV.Field(i).Type.String()
		if name, err := encodeByType(valueV.Field(i), typeV.Field(i).Type.Name()); err != nil { // TODO Encode 无法识别复杂类型
			b = nil
			err = errors.New("encode field name failed, error: " + err.Error())
			return b, err
		} else {
			b = append(b, name...)
		}
	}

	return b, nil
}

func encodeByType(v reflect.Value, t string) ([]byte, error) {
	switch t {
	case "string":
		return Encode(v.String())
	case "int":
		return Encode(int(v.Int()))
	case "int32":
		return Encode(int32(v.Int()))
	case "int64":
		return Encode(v.Int())
	case "bool":
		return Encode(v.Bool())
	case "[]byte":
		return Encode(v.Bytes())
	case "float64":
		return Encode(v.Float())
	case "nil":
		return Encode(nil)
	default:
		return nil, errors.New("unsupport type in object...")
	}
}
