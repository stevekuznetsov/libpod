package createconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/go-units"
)

// Pod signifies a kernel namespace is being shared
// by a container with the pod it is associated with
const Pod = "pod"

// weightDevice is a structure that holds device:weight pair
type weightDevice struct {
	path   string
	weight uint16
}

func (w *weightDevice) String() string {
	return fmt.Sprintf("%s:%d", w.path, w.weight)
}

// LinuxNS is a struct that contains namespace information
// It implemented Valid to show it is a valid namespace
type LinuxNS interface {
	Valid() bool
}

// IsNS returns if the specified string has a ns: prefix
func IsNS(s string) bool {
	parts := strings.SplitN(s, ":", 2)
	return len(parts) > 1 && parts[0] == "ns"
}

// IsPod returns if the specified string is pod
func IsPod(s string) bool {
	return s == Pod
}

// Valid checks the validity of a linux namespace
// s should be the string representation of ns
func Valid(s string, ns LinuxNS) bool {
	return IsPod(s) || IsNS(s) || ns.Valid()
}

// NS is the path to the namespace to join.
func NS(s string) string {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// validateweightDevice validates that the specified string has a valid device-weight format
// for blkio-weight-device flag
func validateweightDevice(val string) (*weightDevice, error) {
	split := strings.SplitN(val, ":", 2)
	if len(split) != 2 {
		return nil, fmt.Errorf("bad format: %s", val)
	}
	if !strings.HasPrefix(split[0], "/dev/") {
		return nil, fmt.Errorf("bad format for device path: %s", val)
	}
	weight, err := strconv.ParseUint(split[1], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid weight for device: %s", val)
	}
	if weight > 0 && (weight < 10 || weight > 1000) {
		return nil, fmt.Errorf("invalid weight for device: %s", val)
	}

	return &weightDevice{
		path:   split[0],
		weight: uint16(weight),
	}, nil
}

// throttleDevice is a structure that holds device:rate_per_second pair
type throttleDevice struct {
	path string
	rate uint64
}

func (t *throttleDevice) String() string {
	return fmt.Sprintf("%s:%d", t.path, t.rate)
}

// validateBpsDevice validates that the specified string has a valid device-rate format
// for device-read-bps and device-write-bps flags
func validateBpsDevice(val string) (*throttleDevice, error) {
	split := strings.SplitN(val, ":", 2)
	if len(split) != 2 {
		return nil, fmt.Errorf("bad format: %s", val)
	}
	if !strings.HasPrefix(split[0], "/dev/") {
		return nil, fmt.Errorf("bad format for device path: %s", val)
	}
	rate, err := units.RAMInBytes(split[1])
	if err != nil {
		return nil, fmt.Errorf("invalid rate for device: %s. The correct format is <device-path>:<number>[<unit>]. Number must be a positive integer. Unit is optional and can be kb, mb, or gb", val)
	}
	if rate < 0 {
		return nil, fmt.Errorf("invalid rate for device: %s. The correct format is <device-path>:<number>[<unit>]. Number must be a positive integer. Unit is optional and can be kb, mb, or gb", val)
	}

	return &throttleDevice{
		path: split[0],
		rate: uint64(rate),
	}, nil
}

// validateIOpsDevice validates that the specified string has a valid device-rate format
// for device-write-iops and device-read-iops flags
func validateIOpsDevice(val string) (*throttleDevice, error) { //nolint
	split := strings.SplitN(val, ":", 2)
	if len(split) != 2 {
		return nil, fmt.Errorf("bad format: %s", val)
	}
	if !strings.HasPrefix(split[0], "/dev/") {
		return nil, fmt.Errorf("bad format for device path: %s", val)
	}
	rate, err := strconv.ParseUint(split[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rate for device: %s. The correct format is <device-path>:<number>. Number must be a positive integer", val)
	}
	if rate < 0 {
		return nil, fmt.Errorf("invalid rate for device: %s. The correct format is <device-path>:<number>. Number must be a positive integer", val)
	}

	return &throttleDevice{
		path: split[0],
		rate: uint64(rate),
	}, nil
}

func getLoggingPath(opts []string) string {
	for _, opt := range opts {
		arr := strings.SplitN(opt, "=", 2)
		if len(arr) == 2 {
			if strings.TrimSpace(arr[0]) == "path" {
				return strings.TrimSpace(arr[1])
			}
		}
	}
	return ""
}

// parseDevice parses device mapping string to a src, dest & permissions string
func parseDevice(device string) (string, string, string, error) { //nolint
	src := ""
	dst := ""
	permissions := "rwm"
	arr := strings.Split(device, ":")
	switch len(arr) {
	case 3:
		if !validDeviceMode(arr[2]) {
			return "", "", "", fmt.Errorf("invalid device mode: %s", arr[2])
		}
		permissions = arr[2]
		fallthrough
	case 2:
		if validDeviceMode(arr[1]) {
			permissions = arr[1]
		} else {
			if arr[1][0] != '/' {
				return "", "", "", fmt.Errorf("invalid device mode: %s", arr[2])
			}
			dst = arr[1]
		}
		fallthrough
	case 1:
		src = arr[0]
	default:
		return "", "", "", fmt.Errorf("invalid device specification: %s", device)
	}

	if dst == "" {
		dst = src
	}
	return src, dst, permissions, nil
}

// validDeviceMode checks if the mode for device is valid or not.
// Valid mode is a composition of r (read), w (write), and m (mknod).
func validDeviceMode(mode string) bool {
	var legalDeviceMode = map[rune]bool{
		'r': true,
		'w': true,
		'm': true,
	}
	if mode == "" {
		return false
	}
	for _, c := range mode {
		if !legalDeviceMode[c] {
			return false
		}
		legalDeviceMode[c] = false
	}
	return true
}
