package bsv

import (
	"bytes"
	"encoding/binary"
)

func op_push_data(data []byte) []byte {
	l := len(data)
	var psh []byte
	var err error
	if l >= 1 && l <= 75 {
		psh = append([]byte{byte(l)}, data...)
	} else if l >= 76 && l <= 255 {
		psh = append([]byte{0x4c, byte(l)}, data...)
	} else if l >= 256 && l <= 65535 {
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, uint16(l))
		psh = append([]byte{0x4d}, append(buf.Bytes(), data...)...)
	} else if l >= 65536 && l <= 4294967295 {
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, uint32(l))
		psh = append([]byte{0x4d}, append(buf.Bytes(), data...)...)
	} else {
		panic("invalid datalen for push")
	}

	if err != nil {
		panic("error writing binary to buffer")
	}

	return psh
}

const (
	op_false                  = byte(0x00)
	op_one_negate             = byte(0x4f)
	op_true                   = byte(0x51)
	op_nop                    = byte(0x61)
	op_if                     = byte(0x63)
	op_notif                  = byte(0x64)
	op_else                   = byte(0x67)
	op_endif                  = byte(0x68)
	op_verify                 = byte(0x69)
	op_return                 = byte(0x6a)
	op_to_alt_stack           = byte(0x6b)
	op_from_alt_stack         = byte(0x6c)
	op_drop2                  = byte(0x6d)
	op_dup2                   = byte(0x6e)
	op_dup3                   = byte(0x6f)
	op_over2                  = byte(0x70)
	op_rot2                   = byte(0x71)
	op_swap2                  = byte(0x72)
	op_ifdup                  = byte(0x73)
	op_depth                  = byte(0x74)
	op_drop                   = byte(0x75)
	op_dup                    = byte(0x76)
	op_nip                    = byte(0x77)
	op_over                   = byte(0x78)
	op_pick                   = byte(0x79)
	op_roll                   = byte(0x7a)
	op_rot                    = byte(0x7b)
	op_swap                   = byte(0x7c)
	op_tuck                   = byte(0x7d)
	op_cat                    = byte(0x7e)
	op_split                  = byte(0x7f)
	op_num2bin                = byte(0x80)
	op_bin2num                = byte(0x81)
	op_size                   = byte(0x82)
	op_invert                 = byte(0x83)
	op_and                    = byte(0x84)
	op_or                     = byte(0x85)
	op_xor                    = byte(0x86)
	op_equal                  = byte(0x87)
	op_equal_verify           = byte(0x88)
	op_add1                   = byte(0x8b)
	op_sub1                   = byte(0x8c)
	op_negate                 = byte(0x8f)
	op_abs                    = byte(0x90)
	op_not                    = byte(0x91)
	op_zero_notequal          = byte(0x92)
	op_add                    = byte(0x93)
	op_sub                    = byte(0x94)
	op_mul                    = byte(0x95)
	op_div                    = byte(0x96)
	op_mod                    = byte(0x97)
	op_lshift                 = byte(0x98)
	op_rshift                 = byte(0x99)
	op_bool_and               = byte(0x9a)
	op_bool_or                = byte(0x9b)
	op_num_equal              = byte(0x9c)
	op_num_equal_verify       = byte(0x9d)
	op_num_not_equal          = byte(0x9e)
	op_less_than              = byte(0x9f)
	op_greater_than           = byte(0xa0)
	op_less_than_or_equal     = byte(0xa1)
	op_greater_than_or_equal  = byte(0xa2)
	op_min                    = byte(0xa3)
	op_max                    = byte(0xa4)
	op_within                 = byte(0xa5)
	op_ripemd160              = byte(0xa6)
	op_sha1                   = byte(0xa7)
	op_sha256                 = byte(0xa8)
	op_hash160                = byte(0xa9)
	op_hash256                = byte(0xaa)
	op_code_separator         = byte(0xab)
	op_check_sig              = byte(0xac)
	op_check_sig_verify       = byte(0xad)
	op_check_multi_sig        = byte(0xae)
	op_check_multi_sig_verify = byte(0xaf)
)

const (
	sig_all                   = byte(0x41)
	sig_none                  = byte(0x42)
	sig_single                = byte(0x43)
	sig_all_anyone_can_pay    = byte(0xc1)
	sig_none_anyone_can_pay   = byte(0xc2)
	sig_single_anyone_can_pay = byte(0xc3)
)
