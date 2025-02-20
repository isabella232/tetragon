// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

package helpers

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/cilium/cilium-e2e/pkg/e2ecluster/e2ehelpers"
	"github.com/cilium/tetragon/pkg/kernels"
	"github.com/cilium/tetragon/tests/e2e/state"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	clusterName string
)

func init() {
	flag.StringVar(&clusterName, "cluster-name", "tetragon-ci", "Set the name of the k8s cluster being used")
}

// GetClusterName fetches the cluster name configured with -cluster-name.
func GetClusterName() string {
	return clusterName
}

func SetMinKernelVersion() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		client, err := cfg.NewClient()
		if err != nil {
			return ctx, err
		}
		r := client.Resources()

		nodeList := &corev1.NodeList{}
		r.List(ctx, nodeList)
		if len(nodeList.Items) < 1 {
			return ctx, fmt.Errorf("failed to list nodes in cluster")
		}

		var versions []string
		var versionsInt []int64

		for _, node := range nodeList.Items {
			name := node.Status.NodeInfo.MachineID
			kVersion := node.Status.NodeInfo.KernelVersion

			// vendors like to define kernel 4.14.128-foo but
			// everything after '-' is meaningless to us so toss it out
			release := strings.Split(kVersion, "-")
			kVersion = release[0]

			klog.Infof("Node %s has kernel version %s", name, kVersion)

			versions = append(versions, kVersion)
			versionsInt = append(versionsInt, kernels.KernelStringToNumeric(kVersion))
		}

		min := int64(math.MaxInt64)
		var minStr string
		for i := range versions {
			verStr := versions[i]
			verInt := versionsInt[i]

			if verInt < min {
				min = verInt
				minStr = verStr
			}
		}

		return context.WithValue(ctx, state.MinKernelVersion, minStr), nil
	}
}

func GetMinKernelVersion(t *testing.T, testenv env.Environment) string {
	version := new(string)
	feature := features.New("Get Minimum Kernel Version").
		Assess("Lookup Kernel Version", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			if v, ok := ctx.Value(state.MinKernelVersion).(string); ok {
				*version = v
			} else {
				assert.Fail(t, "Failed to get kernel version from ctx. Did setup complete properly?")
			}
			return ctx
		}).Feature()
	testenv.Test(t, feature)
	return *version
}

// RunCommand runs a command in a pod and returns the combined stdout and stderr delimited
// by markers as a byte slice.
func RunCommand(ctx context.Context, client klient.Client, podNamespace, podName, containerName string, cmd string, args ...string) ([]byte, error) {
	argv := append([]string{cmd}, args...)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if err := e2ehelpers.ExecInPod(ctx,
		client,
		podNamespace,
		podName,
		containerName,
		stdout,
		stderr,
		argv); err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", cmd, err)
	}

	var err error
	buff := new(bytes.Buffer)
	buff.WriteString("-------------------- stdout starts here --------------------\n")
	if _, err = buff.ReadFrom(stdout); err != nil {
		klog.ErrorS(err, "error reading stdout", "cmd", cmd)
	}
	buff.WriteString("-------------------- stderr starts here --------------------\n")
	if _, err = buff.ReadFrom(stderr); err != nil {
		klog.ErrorS(err, "error reading stderr", "cmd", cmd)
	}
	buff.WriteString("------------------------------------------------------------\n")

	return buff.Bytes(), nil
}
