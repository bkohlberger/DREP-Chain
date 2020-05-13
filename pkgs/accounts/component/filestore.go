package component

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drep-project/DREP-Chain/common/fileutil"
	"github.com/drep-project/DREP-Chain/crypto"
	"github.com/drep-project/DREP-Chain/types"
	"github.com/drep-project/binary"
)

const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

type FileStore struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

func NewFileStore(keyStoreDir string) FileStore {
	if !fileutil.IsDirExists(keyStoreDir) {
		err := os.Mkdir(keyStoreDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	return FileStore{
		keysDirPath: keyStoreDir,
	}
}

func (fs FileStore) ExportAddrs(auth string) ([]string, error) {
	addrs := make([]string, 0)

	err := fileutil.EachChildFile(fs.keysDirPath, func(path string) (bool, error) {
		index := strings.LastIndex(path, "/")
		path = path[index+1:]
		addrs = append(addrs, path)
		return true, nil

	})

	return addrs, err
}

// GetKey read key in file
func (fs FileStore) GetKey(addr *crypto.CommonAddress, auth string) (*types.Node, error) {
	contents, err := ioutil.ReadFile(fs.JoinPath(addr.Hex()))
	if err != nil {
		return nil, ErrKeyNotFound
	}

	node, err := BytesToCryptoNode(contents, auth)
	if err != nil {
		return nil, err
	}

	//ensure ressult after read and decrypto correct
	if node.Address.Hex() != addr.Hex() {
		return nil, fmt.Errorf("key content mismatch: have address %x, want %x", node.Address, addr)
	}
	return node, nil
}

// store the key in file encrypto
func (fs FileStore) StoreKey(key *types.Node, auth string) error {
	cryptoNode := &CryptedNode{
		Version:      0,
		Data:         key.PrivateKey.Serialize(),
		ChainId:      key.ChainId,
		ChainCode:    key.ChainCode,
		Cipher:       "aes-128-ctr",
		CipherParams: CipherParams{},
		KDFParams: ScryptParams{
			N:     StandardScryptN,
			R:     scryptR,
			P:     StandardScryptP,
			Dklen: scryptDKLen,
		},
	}
	cryptoNode.EncryptData([]byte(auth))
	content, err := binary.Marshal(cryptoNode)
	//content, err := json.Marshal(cryptoNode)
	if err != nil {
		return err
	}
	return writeKeyFile(fs.JoinPath(key.Address.Hex()), content)
}

// ExportKey export all key in file by password
func (fs FileStore) ExportKey(auth string) ([]*types.Node, error) {
	persistedNodes := []*types.Node{}

	err := fileutil.EachChildFile(fs.keysDirPath, func(path string) (bool, error) {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			log.WithField("Msg", err).Error("read key store error ")
			return false, err
		}

		fmt.Println("e 0:", time.Now().Unix(), time.Now().Nanosecond())

		node, err := BytesToCryptoNode(contents, auth)
		if err != nil {
			log.WithField("Msg", err).Error("read key store error ", "Msg", err.Error())
			return false, err
		}
		persistedNodes = append(persistedNodes, node)
		fmt.Println("e 1:", time.Now().Unix(), time.Now().Nanosecond())
		return true, nil

	})
	if err != nil {
		return nil, err
	}
	return persistedNodes, nil
}

// JoinPath return keystore directory
func (fs FileStore) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(fs.keysDirPath, filename)
}

func writeTemporaryKeyFile(file string, content []byte) (string, error) {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return "", err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func writeKeyFile(file string, content []byte) error {
	name, err := writeTemporaryKeyFile(file, content)
	if err != nil {
		return err
	}
	return os.Rename(name, file)
}
