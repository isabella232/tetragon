#!/bin/bash

set -eu -o pipefail


OCIORG=quay.io/lvh-images
ROOTIMG=$OCIORG/root-images
KERNIMG=$OCIORG/kernel-images
CONTAINER_ENGINE=${CONTAINER_ENGINE:-docker}
KERNEL_VERS="$@"
DEST_DIR="tests/vmtests/test-data"

if [ -z "$KERNEL_VERS" ]; then
	echo "Usage: $0 <kver1> <kver2>"
	echo "Example: $0 bpf-next 5.10"
	exit 1
fi

set -x
rm -rf $DEST_DIR
mkdir -p $DEST_DIR/images
$CONTAINER_ENGINE pull $ROOTIMG
x=$($CONTAINER_ENGINE create $ROOTIMG)
$CONTAINER_ENGINE cp $x:/data/images/base.qcow2.zst  $DEST_DIR/images/base.qcow2.zst
zstd --decompress $DEST_DIR/images/base.qcow2.zst
$CONTAINER_ENGINE rm $x

mkdir -p $DEST_DIR/kernels
for ver in $KERNEL_VERS; do
	img="$KERNIMG:$ver"
	$CONTAINER_ENGINE pull $img
	x=$($CONTAINER_ENGINE create $img)
	$CONTAINER_ENGINE cp $x:/data/kernels/$ver  $DEST_DIR/kernels
	$CONTAINER_ENGINE rm $x
done
