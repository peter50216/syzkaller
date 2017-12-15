// Copyright 2016 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

func TestSerializeForExecRandom(t *testing.T) {
	target, rs, iters := initTest(t)
	buf := make([]byte, ExecBufferSize)
	for i := 0; i < iters; i++ {
		p := target.Generate(rs, 10, nil)
		n, err := p.SerializeForExec(buf, i%16)
		if err != nil {
			t.Fatalf("failed to serialize: %v", err)
		}
		_, err = target.DeserializeExec(buf[:n])
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSerializeForExec(t *testing.T) {
	target := initTargetTest(t, "test", "64")
	var (
		dataOffset = target.DataOffset
		ptrSize    = target.PtrSize
	)
	callID := func(name string) uint64 {
		c := target.SyscallMap[name]
		if c == nil {
			t.Fatalf("unknown syscall %v", name)
		}
		return uint64(c.ID)
	}
	tests := []struct {
		prog       string
		serialized []uint64
		decoded    *ExecProg
	}{
		{
			"syz_test()",
			[]uint64{
				callID("syz_test"), 0,
				execInstrEOF,
			},
			&ExecProg{
				Calls: []ExecCall{
					{
						Meta:  target.SyscallMap["syz_test"],
						Index: 0,
					},
				},
				NumVars: 1,
			},
		},
		{
			"syz_test$int(0x1, 0x2, 0x3, 0x4, 0x5)",
			[]uint64{
				callID("syz_test$int"), 5,
				execArgConst, 8, 1, 0, 0,
				execArgConst, 1, 2, 0, 0,
				execArgConst, 2, 3, 0, 0,
				execArgConst, 4, 4, 0, 0,
				execArgConst, 8, 5, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align0(&(0x7f0000000000)={0x1, 0x2, 0x3, 0x4, 0x5})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 2, 1, 0, 0,
				execInstrCopyin, dataOffset + 4, execArgConst, 4, 2, 0, 0,
				execInstrCopyin, dataOffset + 8, execArgConst, 1, 3, 0, 0,
				execInstrCopyin, dataOffset + 10, execArgConst, 2, 4, 0, 0,
				execInstrCopyin, dataOffset + 16, execArgConst, 8, 5, 0, 0,
				callID("syz_test$align0"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align1(&(0x7f0000000000)={0x1, 0x2, 0x3, 0x4, 0x5})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 2, 1, 0, 0,
				execInstrCopyin, dataOffset + 2, execArgConst, 4, 2, 0, 0,
				execInstrCopyin, dataOffset + 6, execArgConst, 1, 3, 0, 0,
				execInstrCopyin, dataOffset + 7, execArgConst, 2, 4, 0, 0,
				execInstrCopyin, dataOffset + 9, execArgConst, 8, 5, 0, 0,
				callID("syz_test$align1"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align2(&(0x7f0000000000)={0x42, {[0x43]}, {[0x44]}})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 1, execArgConst, 2, 0x43, 0, 0,
				execInstrCopyin, dataOffset + 4, execArgConst, 2, 0x44, 0, 0,
				callID("syz_test$align2"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align3(&(0x7f0000000000)={0x42, {0x43}, {0x44}})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 1, execArgConst, 1, 0x43, 0, 0,
				execInstrCopyin, dataOffset + 4, execArgConst, 1, 0x44, 0, 0,
				callID("syz_test$align3"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align4(&(0x7f0000000000)={{0x42, 0x43}, 0x44})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 1, execArgConst, 2, 0x43, 0, 0,
				execInstrCopyin, dataOffset + 4, execArgConst, 1, 0x44, 0, 0,
				callID("syz_test$align4"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align5(&(0x7f0000000000)={{0x42, []}, {0x43, [0x44, 0x45, 0x46]}, 0x47})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 8, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 8, execArgConst, 8, 0x43, 0, 0,
				execInstrCopyin, dataOffset + 16, execArgConst, 2, 0x44, 0, 0,
				execInstrCopyin, dataOffset + 18, execArgConst, 2, 0x45, 0, 0,
				execInstrCopyin, dataOffset + 20, execArgConst, 2, 0x46, 0, 0,
				execInstrCopyin, dataOffset + 22, execArgConst, 1, 0x47, 0, 0,
				callID("syz_test$align5"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$align6(&(0x7f0000000000)={0x42, [0x43]})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 4, execArgConst, 4, 0x43, 0, 0,
				callID("syz_test$align6"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$union0(&(0x7f0000000000)={0x1, @f2=0x2})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 8, 1, 0, 0,
				execInstrCopyin, dataOffset + 8, execArgConst, 1, 2, 0, 0,
				callID("syz_test$union0"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$union1(&(0x7f0000000000)={@f1=0x42, 0x43})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 4, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 8, execArgConst, 1, 0x43, 0, 0,
				callID("syz_test$union1"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$union2(&(0x7f0000000000)={@f1=0x42, 0x43})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 4, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 4, execArgConst, 1, 0x43, 0, 0,
				callID("syz_test$union2"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$array0(&(0x7f0000000000)={0x1, [@f0=0x2, @f1=0x3], 0x4})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 1, 0, 0,
				execInstrCopyin, dataOffset + 1, execArgConst, 2, 2, 0, 0,
				execInstrCopyin, dataOffset + 3, execArgConst, 8, 3, 0, 0,
				execInstrCopyin, dataOffset + 11, execArgConst, 8, 4, 0, 0,
				callID("syz_test$array0"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$array1(&(0x7f0000000000)={0x42, \"0102030405\"})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 1, execArgData, 5, 0x0504030201,
				callID("syz_test$array1"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$array2(&(0x7f0000000000)={0x42, \"aaaaaaaabbbbbbbbccccccccdddddddd\", 0x43})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 2, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 2, execArgData, 16, 0xbbbbbbbbaaaaaaaa, 0xddddddddcccccccc,
				execInstrCopyin, dataOffset + 18, execArgConst, 2, 0x43, 0, 0,
				callID("syz_test$array2"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$end0(&(0x7f0000000000)={0x42, 0x42, 0x42, 0x42})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 1, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 1, execArgConst, 2, 0x4200, 0, 0,
				execInstrCopyin, dataOffset + 3, execArgConst, 4, 0x42000000, 0, 0,
				execInstrCopyin, dataOffset + 7, execArgConst, 8, 0x4200000000000000, 0, 0,
				callID("syz_test$end0"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$end1(&(0x7f0000000000)={0xe, 0x42, 0x1})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 2, 0x0e00, 0, 0,
				execInstrCopyin, dataOffset + 2, execArgConst, 4, 0x42000000, 0, 0,
				execInstrCopyin, dataOffset + 6, execArgConst, 8, 0x0100000000000000, 0, 0,
				callID("syz_test$end1"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$bf0(&(0x7f0000000000)={0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 2, 0x42, 0, 10,
				execInstrCopyin, dataOffset + 8, execArgConst, 8, 0x42, 0, 0,
				execInstrCopyin, dataOffset + 16, execArgConst, 2, 0x42, 0, 5,
				execInstrCopyin, dataOffset + 16, execArgConst, 2, 0x42, 5, 6,
				execInstrCopyin, dataOffset + 20, execArgConst, 4, 0x42, 0, 15,
				execInstrCopyin, dataOffset + 24, execArgConst, 2, 0x42, 0, 11,
				execInstrCopyin, dataOffset + 26, execArgConst, 2, 0x4200, 0, 11,
				execInstrCopyin, dataOffset + 28, execArgConst, 1, 0x42, 0, 0,
				callID("syz_test$bf0"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$bf1(&(0x7f0000000000)={{0x42, 0x42, 0x42}, 0x42})",
			[]uint64{
				execInstrCopyin, dataOffset + 0, execArgConst, 4, 0x42, 0, 10,
				execInstrCopyin, dataOffset + 0, execArgConst, 4, 0x42, 10, 10,
				execInstrCopyin, dataOffset + 0, execArgConst, 4, 0x42, 20, 10,
				execInstrCopyin, dataOffset + 4, execArgConst, 1, 0x42, 0, 0,
				callID("syz_test$bf1"), 1, execArgConst, ptrSize, dataOffset, 0, 0,
				execInstrEOF,
			},
			nil,
		},
		{
			"syz_test$res1(0xffff)",
			[]uint64{
				callID("syz_test$res1"), 1, execArgConst, 4, 0xffff, 0, 0,
				execInstrEOF,
			},
			nil,
		},
	}

	buf := make([]byte, ExecBufferSize)
	for i, test := range tests {
		i, test := i, test
		t.Run(fmt.Sprintf("%v:%v", i, test.prog), func(t *testing.T) {
			p, err := target.Deserialize([]byte(test.prog))
			if err != nil {
				t.Fatalf("failed to deserialize prog %v: %v", i, err)
			}
			n, err := p.SerializeForExec(buf, i%16)
			if err != nil {
				t.Fatalf("failed to serialize: %v", err)
			}
			w := new(bytes.Buffer)
			binary.Write(w, binary.LittleEndian, test.serialized)
			data := buf[:n]
			if !bytes.Equal(data, w.Bytes()) {
				got := make([]uint64, len(data)/8)
				binary.Read(bytes.NewReader(data), binary.LittleEndian, &got)
				t.Logf("want: %v", test.serialized)
				t.Logf("got:  %v", got)
				t.Fatalf("mismatch")
			}
			decoded, err := target.DeserializeExec(data)
			if err != nil {
				t.Fatal(err)
			}
			if test.decoded != nil && !reflect.DeepEqual(decoded, *test.decoded) {
				t.Logf("want: %#v", *test.decoded)
				t.Logf("got:  %#v", decoded)
				t.Fatalf("decoded mismatch")
			}
		})
	}
}
