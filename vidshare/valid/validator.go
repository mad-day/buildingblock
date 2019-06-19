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



package valid

import (
	"github.com/mad-day/buildingblock/vidshare"
	shell "github.com/ipfs/go-ipfs-api"
	
	"github.com/tendermint/tendermint/abci/types"
	"regexp"
)

var (
	// "QmTp2hEo8eXRp6wg7jXv1BLCMh5a4F3B7buAUZNZUu772j/28dh743dh87d3h983h8dh8"
	tx_fmt = regexp.MustCompile(`^([^\s\/]+)\/([^\s\/]+)$`)
)

const (
	Code_OK = 0
	Code_Failed = 1
)

type Validator func(mmd *vidshare.MediaMeta) bool

type Event struct{
	Meta *vidshare.MediaMeta
	Ciph *vidshare.Crypto
	Id  string
	Pwd string
	
	Phantom bool // True, if this event does not exist (eg. CheckTx)
}
type EventChan chan Event
func (ec EventChan) Issue(ev Event) {
	if ec==nil { return }
	select {
	case ec <- ev:
	default:
	}
}

/*
The Application layer that filters passed transactions for validity.
Transactions are formatted as:

	tx := []byte(  IpfsHash + "/" + EncryptionPassword  )

Upon validation, the layer pulls the vidshare-object from IPFS, and
validates it (including the decryption password). Aside from existence
and parse-validity, the metadata passes through an array of validators
(.Validators-field) which can decide, whether a media object is Valid
or not.

Finally, if passed, it (optionally) sends the ipfs-hash, metadata and
password through an event channel so that another thread (or agent)
can perform further processing asynchronously.

There is one potential source of Non-Determinism: It uses IPFS.
*/
type AppLayer struct {
	types.Application
	Sh         *shell.Shell
	Validators []Validator
	Evs        EventChan
}
func (a *AppLayer) processTx(tx []byte, pha bool) (bool,string) {
	sm := tx_fmt.FindSubmatch(tx)
	if len(sm)!=3 { return false,"" }
	id := string(sm[1])
	pwd := sm[2]
	
	cr := vidshare.MakeCrypto(pwd)
	
	nmd,err := vidshare.Decode(a.Sh,cr,id)
	if err!=nil { return false,err.Error() }
	
	for _,v := range a.Validators {
		if !v(nmd) { return false,"" }
	}
	
	a.Evs.Issue(Event{nmd,cr,id,string(pwd),pha})
	
	return true,""
}
func (a *AppLayer) CheckTx(tx []byte) types.ResponseCheckTx {
	if ok,log := a.processTx(tx,true); !ok {
		return types.ResponseCheckTx{Code:Code_Failed,Log:log}
	}
	return a.Application.CheckTx(tx)
}
func (a *AppLayer) DeliverTx(tx []byte) types.ResponseDeliverTx {
	if ok,log := a.processTx(tx,false); !ok {
		return types.ResponseDeliverTx{Code:Code_Failed,Log:log}
	}
	return a.Application.DeliverTx(tx)
}



// ##
