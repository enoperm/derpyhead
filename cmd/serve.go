package cmd

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"go4.org/mem"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/key"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Provide just enough of a tailscaled.sock to allow derper to work",
	Run: func(cmd *cobra.Command, args []string) {
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

func init() {
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func respondPeerIds(w http.ResponseWriter, req *http.Request) {
	status, err := fetchStatus()
	if err != nil {
		log.Println(err)
		os.Stdout.WriteString(err.Error())
		return
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(status)
}

func fetchStatus() (status *ipnstate.Status, err error) {
	db, err := sql.Open("sqlite3", appConfig.sourceDatabase)
	if err != nil {
		return
	}
	defer db.Close()

	sb := ipnstate.StatusBuilder{}
	rows, err := db.Query(`
		SELECT machines.node_key, namespaces.name AS namespace_name
		FROM machines, namespaces
		WHERE machines.namespace_id = namespaces.id
	`)

	switch {
	case err == sql.ErrNoRows:
		err = nil
		fallthrough
	case err != nil:
		return
	}

	type record struct {
		nodeKey   string
		namespace string
	}

	dummyStatus := &ipnstate.PeerStatus{}
	process := func(nodeKey, namespace string) error {
		if appConfig.excludeRegex != nil && appConfig.excludeRegex.MatchString(namespace) {
			return nil
		}

		nodeKeyAssHex, err := hex.DecodeString(nodeKey)
		if err != nil {
			return err
		}
		k := key.NodePublicFromRaw32(mem.B(nodeKeyAssHex))
		sb.AddPeer(k, dummyStatus)
		return nil
	}

	for rows.Next() {
		nodeKey, namespace := new(string), new(string)
		rows.Scan(nodeKey, namespace)
		if appConfig.includeRegex == nil || appConfig.includeRegex.MatchString(*namespace) {
			err = process(*nodeKey, *namespace)
			if err != nil {
				return nil, err
			}
		}
	}

	status = sb.Status()
	var nullKey [32]byte

	selfKey := key.NodePublicFromRaw32(mem.B(nullKey[:]))
	status.Self = &ipnstate.PeerStatus{
		PublicKey: selfKey,
	}
	return
}
