/*
Copyright (c) 2019 mad-day

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/


package hashblock

import "github.com/vmihailenco/msgpack"
import "golang.org/x/crypto/blake2b"
import "fmt"

type Tx []byte

type Block struct {
	//_msgpack struct{} `msgpack:",asArray"`
	Index     string `msgpack:"i,omitempty"`
	Timestamp string `msgpack:"t,omitempty"`
	Genesis   []byte `msgpack:"g,omitempty"`
	Txs       []Tx   `msgpack:"t,omitempty"`
	PrevHash  []byte `msgpack:"p,omitempty"`
}
func (blk *Block) String() string {
	return fmt.Sprintf("Block[%d '%s' %q %q #%x]",blk.Index,blk.Timestamp,blk.Genesis,blk.Txs,blk.PrevHash)
}

func Encode(blk *Block) ([]byte,error) {
	if blk==nil { return nil,fmt.Errorf("No block data") }
	return msgpack.Marshal(blk)
}
func Hash(data,blake []byte) []byte {
	hsh := blake2b.Sum256(data)
	return append(blake[:0],hsh[:]...)
}


