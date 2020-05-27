package iden3mobile

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	zktypes "github.com/iden3/go-circom-prover-verifier/types"
	idenpubonchainlocal "github.com/iden3/go-iden3-core/components/idenpubonchain/local"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type config struct {
	Web3Url             string `yaml:"web3Url"`
	IssuerUrl           string `yaml:"issuerUrl"`
	VerifierUrl         string `yaml:"verifierUrl"`
	VerifierAttempts    int    `yaml:"verifierAttempts"`
	VerifierRetryPeriod int    `yaml:"verifierRetryPeriod"`
	HolderTicketPeriod  int    `yaml:"holderTicketPeriod"`
}

var c config
var rmDirs []string
var idenPubOnChain *idenpubonchainlocal.IdenPubOnChain

type TimeBlock struct {
	timeNow  int64
	blockNow uint64
	rw       sync.RWMutex
}

func (tb *TimeBlock) SetTime(t int64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.timeNow = t
}

func (tb *TimeBlock) SetBlock(n uint64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.blockNow = n
}

func (tb *TimeBlock) AddTime(t int64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.timeNow += t
}

func (tb *TimeBlock) AddBlock(n uint64) {
	tb.rw.Lock()
	defer tb.rw.Unlock()
	tb.blockNow += n
}

func (tb *TimeBlock) Time() time.Time {
	tb.rw.RLock()
	defer tb.rw.RUnlock()
	return time.Unix(tb.timeNow, 0)
}

func (tb *TimeBlock) Block() uint64 {
	tb.rw.RLock()
	defer tb.rw.RUnlock()
	return tb.blockNow
}

var timeBlock TimeBlock

func TestMain(m *testing.M) {
	c = config{
		Web3Url:             "xxx",
		IssuerUrl:           "http://188.166.70.93:6100/api/unstable/",
		VerifierUrl:         "http://188.166.70.93:6200/api/unstable/",
		VerifierAttempts:    5,
		VerifierRetryPeriod: 6,
		HolderTicketPeriod:  1000,
	}
	idenPubOnChain = idenpubonchainlocal.New(
		timeBlock.Time,
		timeBlock.Block,
		&zktypes.Vk{},
	)
	// Create a tmp directory to store test files
	// Run tests
	result := m.Run()
	// Remove tmp directory
	for _, dir := range rmDirs {
		os.RemoveAll(dir)
	}
	os.Exit(result)
}

type testEventHandler struct{}

func (teh *testEventHandler) Send(ev *Event) {
	log.Info("Event received: ", ev.TicketId)
}

// NewIdentityTest is like NewIdentity but uses a local implementation of the smart contract in idenPubOnChain
func NewIdentityTest(storePath, sharedStorePath, pass, web3Url string, checkTicketsPeriodMilis int, extraGenesisClaims *BytesArray, s Sender) (*Identity, error) {
	if s == nil {
		s = &testEventHandler{}
	}
	// TODO: tmpIdenPubOnChain ==> idenPubOnChain
	tmpIdenPubOnChain, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		panic(err)
	}
	return newIdentity(storePath, sharedStorePath, pass, tmpIdenPubOnChain, checkTicketsPeriodMilis, extraGenesisClaims, s)
}

// NewIdentityTestLoad is like NewIdentityLoad but uses a local implementation of the smart contract in idenPubOnChain
func NewIdentityTestLoad(storePath, sharedStorePath, pass, web3Url string, checkTicketsPeriodMilis int, s Sender) (*Identity, error) {
	if s == nil {
		s = &testEventHandler{}
	}
	// TODO: tmpIdenPubOnChain ==> idenPubOnChain
	tmpIdenPubOnChain, err := loadIdenPubOnChain(web3Url)
	if err != nil {
		panic(err)
	}
	return newIdentityLoad(storePath, sharedStorePath, pass, tmpIdenPubOnChain, checkTicketsPeriodMilis, s)
}

func TestNewIdentity(t *testing.T) {
	// New identity without extra claims
	sharedDir, err := ioutil.TempDir("", "shared")
	require.Nil(t, err)
	rmDirs = append(rmDirs, sharedDir)
	dir1, err := ioutil.TempDir("", "identityTest")
	rmDirs = append(rmDirs, dir1)
	require.Nil(t, err)
	id, err := NewIdentityTest(dir1, sharedDir, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), nil)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
	// Error when creating new identity on a non empty dir
	_, err = NewIdentityTest(dir1, sharedDir, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, NewBytesArray(), nil)
	require.Error(t, err)
	// Load identity
	id, err = NewIdentityTestLoad(dir1, sharedDir, "pass_TestNewIdentity", c.Web3Url, c.HolderTicketPeriod, nil)
	require.Nil(t, err)
	// Stop identity
	id.Stop()
}
