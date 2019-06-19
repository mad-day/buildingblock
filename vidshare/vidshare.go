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



package vidshare

import (
	"io"
	shell "github.com/ipfs/go-ipfs-api"
	
	"fmt"
	"bytes"
	"encoding/json"
	"errors"
	
	"crypto/cipher"
	"crypto/md5"
	"crypto/aes"
)

var (
	//X = errors.New("X")
	
	// Too many (or too few) sources for given MimeTypes-array. (ZIP-mismatch)
	ErrTooManyFewSources = errors.New("Too many/few sources for given MimeTypes")
)

type MediaMeta struct {
	HtmlType   string `json:"h,omitempty"` // "img", "audio", "video"
	
	MimeTypes  []string `json:"m,omitempty"` // MimeType for each <source> -tag
	
	Title      string `json:"t,omitempty"` // Title
	VideoDescr string `json:"descr,omitempty"` // Video Description
	
	MetaTag    [][2]string `json:"meta,omitempty"` // Metadata tags.
}

type Crypto struct {
	aes cipher.Block
}
func (c *Crypto) Derive(s string) cipher.Stream {
	niv := new([16]byte)
	*niv = md5.Sum([]byte(s))
	c.aes.Encrypt(niv[:],niv[:])
	return cipher.NewCTR(c.aes,niv[:])
}
func rcrypt(r io.Reader,s cipher.Stream) io.Reader {
	return cipher.StreamReader{S:s,R:r}
}
func MakeCrypto(password []byte) *Crypto {
	niv := new([16]byte)
	*niv = md5.Sum(password)
	
	ciph,err := aes.NewCipher(niv[:])
	if err!=nil { panic(err) }
	return &Crypto{aes:ciph}
}

func Upload(sh *shell.Shell, c *Crypto, m *MediaMeta, sources ...io.Reader) (string,error) {
	if len(m.MimeTypes)!=len(sources) { return "",ErrTooManyFewSources }
	b,err := json.Marshal(m)
	if err!=nil { return "",err }
	
	dir, err := sh.NewObject("unixfs-dir")
	if err!=nil { return "",err }
	
	name := "metadata"
	id,err := sh.Add(rcrypt(bytes.NewReader(b),c.Derive(name)))
	if err!=nil { return "",err }
	
	dir,err = sh.PatchLink(dir,name,id,true)
	if err!=nil { return "",err }
	
	for i,src := range sources {
		name = fmt.Sprint(i)
		id,err = sh.Add(rcrypt(src,c.Derive(name)))
		if err!=nil { return "",err }
		
		dir,err = sh.PatchLink(dir,name,id,true)
		if err!=nil { return "",err }
	}
	
	return dir,nil
}

func Decode(sh *shell.Shell, c *Crypto, id string) (md *MediaMeta,_ error) {
	rec,err := sh.Cat(fmt.Sprintf("/ipfs/%s/metadata",id))
	if err!=nil { return nil,err }
	defer rec.Close()
	r := rcrypt(rec,c.Derive("metadata"))
	md = new(MediaMeta)
	err = json.NewDecoder(r).Decode(md)
	if err!=nil { return nil,err }
	return
}

type combo struct{
	io.Closer
	io.Reader
}
func ReadFile(sh *shell.Shell, c *Crypto, id string, name string) (io.ReadCloser,error) {
	rec,err := sh.Cat(fmt.Sprintf("/ipfs/%s/%s",id,name))
	if err!=nil { return nil,err }
	r := rcrypt(rec,c.Derive(name))
	return &combo{rec,r},nil
}

// ##
