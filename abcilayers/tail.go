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


/*
Implements generic layers for building ABCI-applications.
*/
package abcilayers

import (
	"github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	
	"github.com/tendermint/tendermint/crypto/tmhash"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
)

var bE = binary.BigEndian


type info struct {
	Hash []byte
	Heigth int64
}

var infokey = []byte("info")
var bcseedkey = []byte("bcseed")

func emptyTruncated() []byte { return make([]byte,0,20) }


/*
An ABCI-application layer, that calculates a block-hash upon commit and implements
the Info-method correctly (returning the latest Commit block height and app hash).
It also hashes all transactions, that are successfully submitted through DeliverTx.
*/
type HashCounter struct {
	types.Application
	DB dbm.DB
	
	hash []byte
	h1,h2 int64
	digest hash.Hash
}

func (t *HashCounter) OnStart() error {
	
	if t.DB.Has(infokey) {
		i := new(info)
		err := json.Unmarshal(t.DB.Get(infokey),i)
		if err!=nil { return err }
		t.hash = i.Hash
		t.h1 = i.Heigth
	}
	
	t.newDigest()
	
	return nil
}

func (t *HashCounter) newDigest() {
	t.digest = tmhash.NewTruncated()
	if len(t.hash)!=0 {
		t.digest.Write(t.hash)
	}
}
func (t *HashCounter) InitChain(q types.RequestInitChain) (a types.ResponseInitChain) {
	base := tmhash.NewTruncated()
	
	if t.DB.Has(bcseedkey) {
		base.Write(t.DB.Get(bcseedkey))
	}
	
	if len(q.AppStateBytes)!=0 {
		base.Write(q.AppStateBytes)
	}
	t.hash = base.Sum(emptyTruncated())
	t.h1 = 0
	
	t.newDigest()
	
	return t.Application.InitChain(q)
}

func (t *HashCounter) EndBlock(q types.RequestEndBlock) types.ResponseEndBlock {
	t.h2 = q.Height
	return t.Application.EndBlock(q)
}
func (t *HashCounter) Commit() types.ResponseCommit {
	t.Application.Commit()
	
	fmt.Fprintf(t.digest,"{%x}",t.h2)
	t.h1 = t.h2
	t.hash = t.digest.Sum(emptyTruncated())
	
	t.newDigest()
	
	return types.ResponseCommit{Data:t.hash}
}
func (t *HashCounter) Info(q types.RequestInfo) (a types.ResponseInfo) {
	a = t.Application.Info(q)
	a.LastBlockHeight = t.h1
	a.LastBlockAppHash = t.hash
	return
}
func (t *HashCounter) DeliverTx(tx []byte) (a types.ResponseDeliverTx) {
	a = t.Application.DeliverTx(tx)
	if a.Code==0 {
		t.digest.Write(tx)
	}
	
	return
}


/*
An ABCI-application layer, wraps EndBlock and Commit and conveys both .LastBlockHeight
end .LastBlockAppHash out to the Info-method.

This Method is only suited for transient, in-memory Applications.
*/
type HashMemory struct {
	types.Application
	
	hash []byte
	h1,h2 int64
}

func (t *HashMemory) EndBlock(q types.RequestEndBlock) types.ResponseEndBlock {
	t.h2 = q.Height
	return t.Application.EndBlock(q)
}
func (t *HashMemory) Commit() types.ResponseCommit {
	resp := t.Application.Commit()
	
	t.hash = resp.Data
	t.h1 = t.h2
	
	return types.ResponseCommit{Data:t.hash}
}
func (t *HashMemory) Info(q types.RequestInfo) (a types.ResponseInfo) {
	a = t.Application.Info(q)
	a.LastBlockHeight = t.h1
	a.LastBlockAppHash = t.hash
	return
}


// ##
