// Code obtained from https://github.com/etcd-io/etcd/blob/client/v3.5.1/tools/etcd-dump-db/main.go
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

var (
	rootCommand = &cobra.Command{
		Use:   "funny-ip-etcd-detector",
		Short: "funny-ip-etcd-detector inspects etcd db files for finding IPv4 addresses with leading zeros.",
	}

	iterateBucketCommand = &cobra.Command{
		Use:   "find-ips [data dir or db file path]",
		Short: "find-ips lists IPv4 addressess with leading zeroes that will be rejected since golang 1.17 (ref: golang/go#30999).",
		Run:   iterateBucketCommandFunc,
	}
)

var iterateBucketDecode bool
var matchAll bool
var debug bool
var iterateBucketLimit uint64
var flockTimeout time.Duration

func init() {
	rootCommand.PersistentFlags().DurationVar(&flockTimeout, "timeout", 10*time.Second, "time to wait to obtain a file lock on db file, 0 to block indefinitely")
	iterateBucketCommand.PersistentFlags().Uint64Var(&iterateBucketLimit, "limit", 0, "max number of key-value pairs to iterate (0< to iterate all)")
	iterateBucketCommand.PersistentFlags().BoolVar(&iterateBucketDecode, "decode", true, "true to decode Protocol Buffer encoded data")
	iterateBucketCommand.PersistentFlags().BoolVar(&matchAll, "match-all", false, "true to match all IPv4 addresses")
	iterateBucketCommand.PersistentFlags().BoolVar(&debug, "debug", false, "dump all key values")

	rootCommand.AddCommand(iterateBucketCommand)

}

func main() {
	if err := rootCommand.Execute(); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}

func iterateBucketCommandFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		log.Fatalf("Must provide 1 arguments (got %v)", args)
	}
	dp := args[0]
	if !strings.HasSuffix(dp, "db") {
		dp = filepath.Join(snapDir(dp), "db")
	}
	if !existFileOrDir(dp) {
		log.Fatalf("%q does not exist", dp)
	}
	// etcd-dump-db  list-bucket snapshot.db
	// Only interested in the bucket "key"
	bucket := "key"
	err := iterateBucket(dp, bucket, iterateBucketLimit, iterateBucketDecode)
	if err != nil {
		log.Fatal(err)
	}
}

func snapDir(dataDir string) string {
	return filepath.Join(dataDir, "member", "snap")
}

type revision struct {
	main int64
	sub  int64
}

func bytesToRev(bytes []byte) revision {
	return revision{
		main: int64(binary.BigEndian.Uint64(bytes[0:8])),
		sub:  int64(binary.BigEndian.Uint64(bytes[9:])),
	}
}

func iterateBucket(dbPath, bucket string, limit uint64, decode bool) (err error) {
	// https://github.com/golang/go/issues/30999
	// https://www.oreilly.com/library/view/regular-expressions-cookbook/9781449327453/ch08s16.html
	// Accurate regex to check for an IP address, allowing leading zeros:
	re := regexp.MustCompile(`(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`)

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: flockTimeout})
	if err != nil {
		return fmt.Errorf("failed to open bolt DB %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("got nil bucket for %s", bucket)
		}

		c := b.Cursor()

		// iterate in reverse order (use First() and Next() for ascending order)
		var ipError error
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			key, value := k, v

			if decode {
				rev := bytesToRev(k)
				var kv mvccpb.KeyValue
				if err := kv.Unmarshal(v); err != nil {
					return err
				}
				key = kv.Key
				value = kv.Value
				if debug {
					fmt.Printf("rev=%+v, value=[key %q | val %q | created %d | mod %d | ver %d]\n", rev, string(kv.Key), string(kv.Value), kv.CreateRevision, kv.ModRevision, kv.Version)
				}
			} else if debug {
				fmt.Printf("key %q | val %q\n", key, value)
			}

			var result [][]byte
			result = append(result, re.FindAll(key, -1)...)
			result = append(result, re.FindAll(value, -1)...)
			if matchAll && len(result) > 0 {
				fmt.Printf("IPv4 addresses found %q on key: %q\n", result, key)
			}

			var invalidIPs []string
			for _, ip := range result {
				if !parseIPv4(string(ip)) {
					invalidIPs = append(invalidIPs, string(ip))
				}
			}
			if len(invalidIPs) > 0 {
				fmt.Printf("WARNING Invalid IPv4 addresses %q on key: %q\n", invalidIPs, key)
				ipError = fmt.Errorf("Invalid IPv4 addresses found")
			}

			limit--
			if limit == 0 {
				break
			}
		}
		return ipError
	})

	return err
}

func existFileOrDir(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// https://github.com/golang/go/blob/3796df1b13c6be62ca28244dcd6121544770e371/src/net/ip.go#L560
// Parse IPv4 address (d.d.d.d), return true if is a valid IP, IPs with leading zeros are not valid
func parseIPv4(s string) bool {
	const IPv4len = 4
	for i := 0; i < IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			return false
		}
		if i > 0 {
			if s[0] != '.' {
				return false
			}
			s = s[1:]
		}
		n, c, ok := dtoi(s)
		if !ok || n > 0xFF {
			return false
		}
		if c > 1 && s[0] == '0' {
			// Reject non-zero components with leading zeroes.
			return false
		}
		s = s[c:]
	}
	if len(s) != 0 {
		return false
	}
	return true
}

// https://github.com/golang/go/blob/c04a32e59a001f0490082619bbe6a36e1e23ef99/src/net/parse.go#L117-L134
// Bigger than we need, not too big to worry about overflow
const big = 0xFFFFFF

// Decimal to integer.
// Returns number, characters consumed, success.
func dtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return big, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}
