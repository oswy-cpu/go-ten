package ethadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

const (
	versionMethod        = "eth/v1/node/version"
	specMethod           = "eth/v1/config/spec"
	genesisMethod        = "eth/v1/beacon/genesis"
	sidecarsMethodPrefix = "eth/v1/beacon/blob_sidecars/"
)

// L1BeaconClient is a high level golang client for the Beacon API.
type L1BeaconClient struct {
	cl           BeaconClient
	pool         *ClientPool[BlobSideCarsFetcher]
	initLock     sync.Mutex
	timeToSlotFn TimeToSlotFn
}

// BeaconClient is a thin wrapper over the Beacon APIs.
type BeaconClient interface {
	NodeVersion(ctx context.Context) (string, error)
	ConfigSpec(ctx context.Context) (APIConfigResponse, error)
	BeaconGenesis(ctx context.Context) (APIGenesisResponse, error)
	BeaconBlobSideCars(ctx context.Context, slot uint64, hashes []IndexedBlobHash) (APIGetBlobSidecarsResponse, error)
}

// BlobSideCarsFetcher is a thin wrapper over the Beacon APIs.
type BlobSideCarsFetcher interface {
	BeaconBlobSideCars(ctx context.Context, slot uint64, hashes []IndexedBlobHash) (APIGetBlobSidecarsResponse, error)
}

// BeaconHTTPClient implements BeaconClient. It provides golang types over the basic Beacon API.
type BeaconHTTPClient struct {
	client  *http.Client
	baseURL string
}

func NewBeaconHTTPClient(client *http.Client, baseURL string) *BeaconHTTPClient {
	return &BeaconHTTPClient{client: client, baseURL: baseURL}
}

func (bc *BeaconHTTPClient) request(ctx context.Context, dest any, reqPath string, reqQuery url.Values) error {
	baseURL, err := url.Parse(bc.baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %w", err)
	}

	reqURL, err := baseURL.Parse(reqPath)
	if err != nil {
		return fmt.Errorf("failed to parse request path: %w", err)
	}

	reqURL.RawQuery = reqQuery.Encode()

	headers := http.Header{}
	headers.Add("Accept", "application/json")

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header = headers

	resp, err := bc.client.Do(req)
	if err != nil {
		return fmt.Errorf("http Get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		errMsg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed request with status %d: %s: %w", resp.StatusCode, string(errMsg), ethereum.NotFound)
	} else if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed request with status %d: %s", resp.StatusCode, string(errMsg))
	}

	err = json.NewDecoder(resp.Body).Decode(dest)
	if err != nil {
		return err
	}

	return nil
}

func (bc *BeaconHTTPClient) NodeVersion(ctx context.Context) (string, error) {
	var resp APIVersionResponse
	if err := bc.request(ctx, &resp, versionMethod, nil); err != nil {
		return "", err
	}
	return resp.Data.Version, nil
}

func (bc *BeaconHTTPClient) ConfigSpec(ctx context.Context) (APIConfigResponse, error) {
	var configResp APIConfigResponse
	if err := bc.request(ctx, &configResp, specMethod, nil); err != nil {
		return APIConfigResponse{}, err
	}
	return configResp, nil
}

func (bc *BeaconHTTPClient) BeaconGenesis(ctx context.Context) (APIGenesisResponse, error) {
	var genesisResp APIGenesisResponse
	if err := bc.request(ctx, &genesisResp, genesisMethod, nil); err != nil {
		return APIGenesisResponse{}, err
	}
	return genesisResp, nil
}

func (bc *BeaconHTTPClient) BeaconBlobSideCars(ctx context.Context, slot uint64, hashes []IndexedBlobHash) (APIGetBlobSidecarsResponse, error) {
	reqPath := path.Join(sidecarsMethodPrefix, strconv.FormatUint(slot, 10))
	var reqQuery url.Values
	//FIXME lookup sidecars by their hash
	//reqQuery = url.Values{}
	//for i := range hashes {
	//	reqQuery.Add("indices", strconv.FormatUint(hashes[i].Index, 10))
	//}
	var resp APIGetBlobSidecarsResponse

	err := bc.request(ctx, &resp, reqPath, reqQuery)

	if err != nil {
		return APIGetBlobSidecarsResponse{}, err
	}
	return resp, nil
}

type ClientPool[T any] struct {
	clients []T
	index   int
}

func NewClientPool[T any](clients ...T) *ClientPool[T] {
	return &ClientPool[T]{
		clients: clients,
		index:   0,
	}
}

func (p *ClientPool[T]) Len() int {
	return len(p.clients)
}

func (p *ClientPool[T]) Get() T {
	return p.clients[p.index]
}

func (p *ClientPool[T]) MoveToNext() {
	p.index += 1
	if p.index == len(p.clients) {
		p.index = 0
	}
}

// NewL1BeaconClient returns a client for making requests to an L1 consensus layer node.
// Fallbacks are optional clients that will be used for fetching blobs. L1BeaconClient will rotate between
// the `cl` and the fallbacks whenever a client runs into an error while fetching blobs.
func NewL1BeaconClient(cl BeaconClient, fallbacks ...BlobSideCarsFetcher) *L1BeaconClient {
	cs := append([]BlobSideCarsFetcher{cl}, fallbacks...)
	return &L1BeaconClient{
		cl:   cl,
		pool: NewClientPool[BlobSideCarsFetcher](cs...)}
}

type TimeToSlotFn func(timestamp uint64) (uint64, error)

// GetTimeToSlotFn returns a function that converts a timestamp to a slot number.
func (cl *L1BeaconClient) GetTimeToSlotFn(ctx context.Context) (TimeToSlotFn, error) {
	cl.initLock.Lock()
	defer cl.initLock.Unlock()
	if cl.timeToSlotFn != nil {
		return cl.timeToSlotFn, nil
	}

	genesis, err := cl.cl.BeaconGenesis(ctx)
	if err != nil {
		return nil, err
	}

	config, err := cl.cl.ConfigSpec(ctx)
	if err != nil {
		return nil, err
	}

	genesisTime := uint64(genesis.Data.GenesisTime)
	secondsPerSlot := uint64(config.Data.SecondsPerSlot)
	if secondsPerSlot == 0 {
		return nil, fmt.Errorf("got bad value for seconds per slot: %v", config.Data.SecondsPerSlot)
	}
	cl.timeToSlotFn = func(timestamp uint64) (uint64, error) {
		if timestamp < genesisTime {
			return 0, fmt.Errorf("provided timestamp (%v) precedes genesis time (%v)", timestamp, genesisTime)
		}
		return (timestamp - genesisTime) / secondsPerSlot, nil
	}
	return cl.timeToSlotFn, nil
}

func (cl *L1BeaconClient) fetchSidecars(ctx context.Context, slot uint64, hashes []IndexedBlobHash) (APIGetBlobSidecarsResponse, error) {
	var errs []error
	for i := 0; i < cl.pool.Len(); i++ {
		f := cl.pool.Get()
		resp, err := f.BeaconBlobSideCars(ctx, slot, hashes)
		if err != nil {
			cl.pool.MoveToNext()
			errs = append(errs, err)
		} else {
			return resp, nil
		}
	}
	return APIGetBlobSidecarsResponse{}, errors.Join(errs...)
}

// GetBlobSidecars fetches blob sidecars that were confirmed in the specified
// L1 block with the given indexed hashes.
// Order of the returned sidecars is guaranteed to be that of the hashes.
// Blob data is not checked for validity.
func (cl *L1BeaconClient) GetBlobSidecars(ctx context.Context, b *types.Header, hashes []IndexedBlobHash) ([]*BlobSidecar, error) {
	if len(hashes) == 0 {
		return []*BlobSidecar{}, nil
	}
	slotFn, err := cl.GetTimeToSlotFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get time to slot function: %w", err)
	}
	slot, err := slotFn(b.Time)
	if err != nil {
		return nil, fmt.Errorf("error in converting ref.Time to slot: %w", err)
	}

	resp, err := cl.fetchSidecars(ctx, slot, hashes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob sidecars for slot %v block %v: %w", slot, b, err)
	}

	apiscs := make([]*APIBlobSidecar, 0, len(hashes))
	// filter and order by hashes
	for _, h := range hashes {
		for _, apisc := range resp.Data {
			if h.Index == uint64(apisc.Index) {
				apiscs = append(apiscs, apisc)
				break
			}
		}
	}

	if len(hashes) != len(apiscs) {
		return nil, fmt.Errorf("expected %v sidecars but got %v", len(hashes), len(apiscs))
	}

	bscs := make([]*BlobSidecar, 0, len(hashes))
	for _, apisc := range apiscs {
		bscs = append(bscs, apisc.BlobSidecar())
	}

	return bscs, nil
}

// FetchBlobs fetches blobs that were confirmed in the specified L1 block with the given indexed
// hashes. The order of the returned blobs will match the order of `hashes`.  Confirms each
// blob's validity by checking its proof against the commitment, and confirming the commitment
// hashes to the expected value. Returns error if any blob is found invalid.
func (cl *L1BeaconClient) FetchBlobs(ctx context.Context, b *types.Header, hashes []IndexedBlobHash) ([]*kzg4844.Blob, error) {
	blobSidecars, err := cl.GetBlobSidecars(ctx, b, hashes)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob sidecars for Block Header %s: %w", b.Hash().Hex(), err)
	}
	return blobsFromSidecars(blobSidecars, hashes)
}

func blobsFromSidecars(blobSidecars []*BlobSidecar, hashes []IndexedBlobHash) ([]*kzg4844.Blob, error) {
	if len(blobSidecars) != len(hashes) {
		return nil, fmt.Errorf("number of hashes and blobSidecars mismatch, %d != %d", len(hashes), len(blobSidecars))
	}

	out := make([]*kzg4844.Blob, len(hashes))
	for i, ih := range hashes {
		sidecar := blobSidecars[i]
		if sidx := uint64(sidecar.Index); sidx != ih.Index {
			return nil, fmt.Errorf("expected sidecars to be ordered by hashes, but got %d != %d", sidx, ih.Index)
		}

		// make sure the blob's kzg commitment hashes to the expected value
		hash := KZGToVersionedHash(kzg4844.Commitment(sidecar.KZGCommitment))
		if hash != ih.Hash {
			return nil, fmt.Errorf("expected hash %s for blob at index %d but got %s", ih.Hash, ih.Index, hash)
		}

		// confirm blob data is valid by verifying its proof against the commitment
		if err := VerifyBlobProof(&sidecar.Blob, kzg4844.Commitment(sidecar.KZGCommitment), kzg4844.Proof(sidecar.KZGProof)); err != nil {
			return nil, fmt.Errorf("blob at index %d failed verification: %w", i, err)
		}
		out[i] = &sidecar.Blob
	}
	return out, nil
}

// GetVersion fetches the version of the Beacon-node.
func (cl *L1BeaconClient) GetVersion(ctx context.Context) (string, error) {
	return cl.cl.NodeVersion(ctx)
}
