#!/bin/bash
#
# SPDX-License-Identifier: AGPL-3.0-only
#
# Copyright (c) 2025 sycured
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, version 3.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
#

#
# Simple benchmark test suite
#
# To install pewpew: go get github.com/bengadbois/pewpew
# To install vegeta: go get github.com/tsenart/vegeta
#
#
set -xeo pipefail

# Default port to listen
port=8088

# Start the server
./bin/imaginary -p $port &>/dev/null &
pid=$!

suite=""
if command -v pewpew 2>&1 >/dev/null
then
  suite=suite_pewpew
elif command -v vegeta if command -v pewpew 2>&1 >/dev/null
then
  suite=suite_vegeta
else
  echo "Please install either betwen pewpew or vegeta to run the benchmark"
  exit 1
fi

suite_pewpew() {
  echo "$1 --------------------------------------"
  pewpew benchmark "http://localhost:$port/$2" -X POST -d 30 --rps 50 --body-file "./testdata/large.jpg"
  sleep 1
}

suite_vegeta() {
  echo "$1 --------------------------------------"
  echo "POST http://localhost:$port/$2" | vegeta attack \
    -duration=30s \
    -rate=50 \
    -body="./testdata/large.jpg" \ | vegeta report
  sleep 1
}

# Run suites
$suite "Crop" "crop?width=800&height=600"
$suite "Resize" "resize?width=200"
#$suite "Rotate" "rotate?rotate=180"
#$suite "Enlarge" "enlarge?width=1600&height=1200"
$suite "Extract" "extract?top=50&left=50&areawidth=200&areaheight=200"

# Kill the server
kill -9 $pid
