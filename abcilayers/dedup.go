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



package abcilayers


import (
	"github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	
	"hash/fnv"
	"bytes"
	"fmt"
)

type Tx2Key func(tx []byte) (key []byte)
func (t Tx2Key) Extract(tx []byte) (key []byte) {
	if t==nil {
		key = tx
	} else {
		key = t(tx)
	}
	if len(key)==0 { key = []byte("NULL") }
	return
}

func db_temp_init(dbp *dbm.DB) {
	if (*dbp)!=nil { return }
	*dbp = dbm.NewMemDB()
}
func db_temp_clear(dbp *dbm.DB) {
	*dbp = dbm.NewMemDB()
}
func dedup_hash(key []byte) []byte {
	h := fnv.New32a()
	h.Write(key)
	// We limit the maximum number of buckets to 16 Million
	return []byte(fmt.Sprintf("dedup%06X",h.Sum32()&0xffffff))
}
func dedup_clone(k []byte) []byte {
	nb := make([]byte,len(k))
	copy(nb,k)
	return nb
}

/*
An ABCI-application layer, that de-duplicates transactions.
From the transaction, a key is extracted. If there is no such key
extraction function, the transaction itself is the key.

The key is hashed with a 32-bit function (FNV1a) and truncated to 24-bit,
so that maximum number of hash buckets is 2²⁴ (~ 16 millions).

In the case of a hash collision in the truncated 24-bit hash space, the
algorithm can fail to detect some duplicate TX-keys, but the algorithm
guarantees, that no TX-keys are falsely detected as duplicates.

The bucket numbers are formatted as hex with the "dedup" prefix, so that
a key looks like "dedupF2A450".
*/
type DedupLayer struct {
	types.Application
	DB dbm.DB
	
	KeyExtractor Tx2Key
	
	mem,buf dbm.DB
}

func (d *DedupLayer) getObj(key []byte,d2p *dbm.DB) []byte {
	db_temp_init(d2p)
	d2 := *d2p
	if d2.Has(key) { return d2.Get(key) }
	return d.DB.Get(key)
}
func (d *DedupLayer) hasdupe(tx []byte,d2p *dbm.DB) (failed bool) {
	key := d.KeyExtractor.Extract(tx)
	h := dedup_hash(key)
	
	if bytes.Equal(d.getObj(h,d2p),key) {
		return true
	}
	return false
}
func (d *DedupLayer) setdupe(tx []byte,d2p *dbm.DB) {
	key := d.KeyExtractor.Extract(tx)
	h := dedup_hash(key)
	(*d2p).Set(h,dedup_clone(key))
}
func (d *DedupLayer) CheckTx(tx []byte) (q types.ResponseCheckTx) {
	if d.hasdupe(tx,&d.mem) { return types.ResponseCheckTx{Code:1} }
	q = d.Application.CheckTx(tx)
	if q.Code==0 { d.setdupe(tx,&d.mem) }
	return
}
func (d *DedupLayer) DeliverTx(tx []byte) (q types.ResponseDeliverTx)  {
	if d.hasdupe(tx,&d.buf) { return types.ResponseDeliverTx{Code:1} }
	q = d.Application.DeliverTx(tx)
	if q.Code==0 { d.setdupe(tx,&d.buf) }
	return
}
func (d *DedupLayer) Commit() (a types.ResponseCommit) {
	a = d.Application.Commit()
	defer db_temp_clear(&d.mem)
	defer db_temp_clear(&d.buf)
	iter := d.buf.Iterator([]byte("dedup\x00"), []byte("dedup\xff"))
	defer iter.Close()
	bat := d.DB.NewBatch()
	defer bat.Close()
	for ; iter.Valid() ; iter.Next() {
		bat.Set(iter.Key(),iter.Value())
	}
	bat.WriteSync()
	return
}

// ##
