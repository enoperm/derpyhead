package cmd

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"go4.org/mem"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/key"
)

var cache struct {
	mutex sync.Mutex
	keys  []key.NodePublic
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Provide just enough of a tailscaled.sock to allow derper to work",
	Run: func(cmd *cobra.Command, args []string) {
		go func() {
			dur, err := time.ParseDuration(appConfig.updateInterval)
			if err != nil {
				log.Fatalf("config: update-interval: %q, %s", appConfig.updateInterval, err)
			}
			ticker := time.NewTicker(dur)
			defer ticker.Stop()

			updateCache()
			for range ticker.C {
				updateCache()
			}
		}()

		http.HandleFunc("/", respondPeerIds)
		os.Remove(appConfig.listenAddr)
		listenSock, err := net.Listen("unix", appConfig.listenAddr)
		if err != nil {
			log.Fatal(err)
		}
		defer listenSock.Close()
		log.Println(http.Serve(listenSock, nil))
	},
}

func updateCache() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, appConfig.keysCommand, appConfig.keysCommandArgs...)
	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("update-cache:", err)
	}

	cmd.Start()

	newKeys := make([]key.NodePublic, 0, 64)
	buffered := bufio.NewScanner(output)
	for buffered.Scan() {
		kstring := buffered.Text()
		if len(kstring) < 1 {
			continue
		}

		nodeKey, err := hex.DecodeString(kstring)
		if err != nil {
			log.Printf("update-cache: read-key: %q: %s", kstring, err)
			return
		}
		k := key.NodePublicFromRaw32(mem.B(nodeKey))
		newKeys = append(newKeys, k)
	}

	err = cmd.Wait()
	if err != nil {
		log.Printf("update-cache: %s", err)
		return
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.keys = newKeys
}

func respondPeerIds(w http.ResponseWriter, req *http.Request) {
	status := buildStatus()
	encoder := json.NewEncoder(w)
	encoder.Encode(status)
}

func buildStatus() (status *ipnstate.Status) {
	sb := ipnstate.StatusBuilder{}

	func() {
		cache.mutex.Lock()
		defer cache.mutex.Unlock()

		for _, k := range cache.keys {
			sb.AddPeer(k, &ipnstate.PeerStatus{})
		}
	}()

	status = sb.Status()
	var nullKey [32]byte

	selfKey := key.NodePublicFromRaw32(mem.B(nullKey[:]))
	status.Self = &ipnstate.PeerStatus{
		PublicKey: selfKey,
	}
	return
}
