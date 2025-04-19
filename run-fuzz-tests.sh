#!/bin/bash

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

FUZZ_TIME=${1:-90s}

OK_COLOR="\033[32;01m"
NO_COLOR="\033[0m"

PACKAGES=$(go list ./... | grep -v /vendor/)

EXIT_CODE=0

for PKG in $PACKAGES; do
  FUZZ_TESTS=$(go test -list='^Fuzz' "$PKG" | grep '^Fuzz')

  if [ -n "$FUZZ_TESTS" ]; then
    while IFS= read -r TEST_NAME; do
      echo -e "${OK_COLOR}Running fuzz test $TEST_NAME${NO_COLOR}"
      go test "$PKG" -run=^$ -fuzz="$TEST_NAME" -fuzztime="$FUZZ_TIME"
      # Capture the exit code
      if [ $? -ne 0 ]; then
        echo "Fuzz test $TEST_NAME failed."
        EXIT_CODE=1
      fi
    done <<< "$FUZZ_TESTS"
  else
    echo "No fuzz tests found in $PKG"
  fi
done

exit $EXIT_CODE