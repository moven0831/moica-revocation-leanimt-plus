//go:build js && wasm

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"runtime"
	"syscall/js"

	"github.com/moven0831/moica-revocation-smt/server/internal/hexenc"
	"github.com/moven0831/moica-revocation-smt/server/internal/leanimt"
	"github.com/moven0831/moica-revocation-smt/server/internal/snapshot"
)

var (
	hasher leanimt.Hasher
	tree   *leanimt.LeanIMTPlus
)

func main() {
	hasher = leanimt.NewPoseidonHasher()

	js.Global().Set("leanimtLoadSnapshot", js.FuncOf(loadSnapshot))
	js.Global().Set("leanimtGenerateProof", js.FuncOf(generateProof))
	js.Global().Set("leanimtVerifyProof", js.FuncOf(verifyProofJS))
	js.Global().Set("leanimtGetMemStats", js.FuncOf(getMemStats))
	js.Global().Set("leanimtRoot", js.FuncOf(rootJS))
	js.Global().Set("leanimtReady", js.ValueOf(true))

	select {}
}

// loadSnapshot(uint8Array) — loads a decompressed v2 binary snapshot into the tree.
func loadSnapshot(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsError("loadSnapshot requires (uint8Array)")
	}
	jsArr := args[0]
	length := jsArr.Get("length").Int()
	buf := make([]byte, length)
	js.CopyBytesToGo(buf, jsArr)

	loaded, _, err := snapshot.ImportBinary(hasher, bytes.NewReader(buf))
	if err != nil {
		return jsError(fmt.Sprintf("import binary: %v", err))
	}
	tree = loaded
	return map[string]any{
		"size":      tree.Size(),
		"depth":     tree.Depth(),
		"leafCount": tree.LeafCount(),
	}
}

func generateProof(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsError("generateProof requires (valueHex)")
	}
	if tree == nil {
		return jsError("tree not loaded — call leanimtLoadSnapshot first")
	}

	valueHex := args[0].String()
	v, ok := new(big.Int).SetString(valueHex, 16)
	if !ok {
		return jsError(fmt.Sprintf("invalid value hex: %s", valueHex))
	}

	p, err := tree.GenerateProof(v)
	if err != nil {
		return jsError(err.Error())
	}
	out, err := json.Marshal(proofToJSON(p))
	if err != nil {
		return jsError(fmt.Sprintf("marshal proof: %v", err))
	}
	return string(out)
}

func verifyProofJS(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsError("verifyProof requires (proofJSON)")
	}
	var jp jsonProof
	if err := json.Unmarshal([]byte(args[0].String()), &jp); err != nil {
		return jsError(fmt.Sprintf("unmarshal proof: %v", err))
	}
	return leanimt.VerifyProof(hasher, jsonToProof(&jp))
}

func rootJS(_ js.Value, _ []js.Value) any {
	if tree == nil {
		return ""
	}
	return hexenc.Encode(tree.Root())
}

func getMemStats(_ js.Value, _ []js.Value) any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	result := map[string]any{
		"alloc":      m.Alloc,
		"totalAlloc": m.TotalAlloc,
		"sys":        m.Sys,
		"heapInuse":  m.HeapInuse,
		"heapAlloc":  m.HeapAlloc,
		"numGC":      m.NumGC,
	}
	out, _ := json.Marshal(result)
	return string(out)
}

func jsError(msg string) any {
	return js.Global().Get("Error").New(msg)
}

type jsonLeaf struct {
	Value     string `json:"value"`
	NextValue string `json:"nextValue"`
}

type jsonProof struct {
	ProofType int      `json:"proofType"`
	Root      string   `json:"root"`
	Value     string   `json:"value"`
	Leaf      jsonLeaf `json:"leaf"`
	LeafIndex uint64   `json:"leafIndex"`
	Siblings  []string `json:"siblings"`
}

func decodeOrZero(s string) *big.Int {
	n, err := hexenc.Decode(s)
	if err != nil {
		return new(big.Int)
	}
	return n
}

func proofToJSON(p *leanimt.Proof) *jsonProof {
	jp := &jsonProof{
		ProofType: int(p.ProofType),
		Root:      hexenc.Encode(p.Root),
		Value:     hexenc.Encode(p.Value),
		Leaf:      jsonLeaf{Value: hexenc.Encode(p.Leaf.Value), NextValue: hexenc.Encode(p.Leaf.NextValue)},
		LeafIndex: p.LeafIndex,
		Siblings:  hexenc.EncodeSlice(p.Siblings),
	}
	return jp
}

func jsonToProof(jp *jsonProof) *leanimt.Proof {
	p := &leanimt.Proof{
		ProofType: leanimt.ProofType(jp.ProofType),
		Root:      decodeOrZero(jp.Root),
		Value:     decodeOrZero(jp.Value),
		Leaf:      leanimt.IndexedLeaf{Value: decodeOrZero(jp.Leaf.Value), NextValue: decodeOrZero(jp.Leaf.NextValue)},
		LeafIndex: jp.LeafIndex,
		Siblings:  make([]*big.Int, len(jp.Siblings)),
	}
	for i, s := range jp.Siblings {
		p.Siblings[i] = decodeOrZero(s)
	}
	return p
}
