package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/dgraph-io/badger"
)

// Block ...
type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

// BlockChain ...
// type BlockChain struct {
// 	Blocks   []*Block
// 	Database *badger.DB
// }

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// DeriveHash ...
// func (b *Block) DeriveHash() {
// 	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
// 	hash := sha256.Sum256(info)
// 	b.Hash = hash[:]

// }

// CreateBlock ...
func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	pow := NewProof(block)

	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce
	// block.DeriveHash()

	return block
}

func Handle(err error) {

	if err != nil {
		log.Fatal(err)
	}
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}
	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)

		var encodedBlock []byte
		err = item.Value(func(value []byte) error {
			encodedBlock = value
			return nil
		})

		block = Deserialize(encodedBlock)

		return err
	})

	Handle(err)
	iter.CurrentHash = block.PrevHash
	return block
}

// AddBlock ...
func (chain *BlockChain) AddBlock(data string) {
	// prevBlock := chain.Blocks[len(chain.Blocks)-1]
	// new := CreateBlock(data, prevBlock.Hash)
	// chain.Blocks = append(chain.Blocks, new)

	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err2 := txn.Get([]byte("lh"))

		Handle(err2)

		// lastHash, err2 = item.ValueCopy(lastHash)
		err2 = item.Value(func(value []byte) error {
			lastHash = value
			return nil
		})

		return err2
	})

	Handle(err)
	newBlock := CreateBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		return err
	})

}

// Genesis ...
func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

// InitBlockChain ...
func InitBlockChain() *BlockChain {
	var lastHash []byte
	opts := badger.DefaultOptions("./db")

	fmt.Println("Enter Init Blockchain")

	db, err := badger.Open(opts)

	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {

			fmt.Println(" Cannot get found last hash")

			fmt.Println("No existing blockchain found")
			genesis := Genesis()
			fmt.Println(" Genesis block Proved")

			err = txn.Set(genesis.Hash, genesis.Serialize())

			Handle(err)

			err = txn.Set([]byte("lh"), genesis.Hash)

			lastHash = genesis.Hash

		} else {
			item, err := txn.Get([]byte("lh"))

			err = item.Value(func(value []byte) error {
				lastHash = value
				return nil
			},
			)
			return err
		}

		return err
	})

	blockchain := BlockChain{lastHash, db}
	return &blockchain
	// return &BlockChain{[]*Block{Genesis()}}
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)

	if err != nil {
		log.Panic(err)
	}

	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)

	if err != nil {
		log.Panic(err)
	}

	return &block
}
