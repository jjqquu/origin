#!/bin/bash

# This script builds all images locally except the base and release images,
# which are handled by hack/build-base-images.sh.

# NOTE:  you only need to run this script if your code changes are part of
# any images OpenShift runs internally such as origin-sti-builder, origin-docker-builder,
# origin-deployer, etc.

# Create link to file if the FS supports hardlinks, otherwise copy the file
OS_ROOT=$(dirname "${BASH_SOURCE}")/..
source "${OS_ROOT}/hack/common.sh"

function ln_or_cp {
  local src_file=$1
  local dst_dir=$2
  echo "cp -pf ${src_file} ${dst_dir}"
  cp -pf "${src_file}" "${dst_dir}"
}

imagedir="${OS_OUTPUT_BINPATH}/linux/amd64"

# Link or copy image binaries to the appropriate locations.
ln_or_cp "${imagedir}/openshift"       images/origin/bin/
ln_or_cp "${imagedir}/openshift"       images/siteagent/bin/
ln_or_cp "${imagedir}/openshift"       images/marathon-deployer/bin/

# builds an image and tags it two ways - with latest, and with the release tag
# 2 modes: debug v.s. non-debug -- with/without diagnosis tools
#          cache v.s. no-cache  -- docker build with cache enabled/disabled
function image {
  echo "--- $1 ---"
  echo "docker build -t $1:latest $2"
  docker build -t $1:latest $2
  echo
  echo
}

function image_nocache {
  echo "--- $1 ---"
  echo "docker build --no-cache -t $1:latest $2"
  docker build --no-cache -t $1:latest $2
  echo
  echo
}

function image_debug {
  echo "--- $2 ---"
  echo "docker build -t $1:latest -f $2/Dockerfile.debug $2"
  docker build -t $1:latest -f $2/Dockerfile.debug $2
  echo
  echo
}

function image_debug_nocache {
  echo "--- $2 ---"
  echo "docker build --no-cache -t $1:latest -f $2/Dockerfile.debug $2"
  docker build --no-cache -t $1:latest -f $2/Dockerfile.debug $2
  echo
  echo
}

image_debug 		openshift/origin                       images/origin
image_debug_nocache 	openshift/origin-siteagent             images/siteagent
image_debug_nocache 	openshift/origin-marathon-deployer     images/marathon-deployer


