#!/bin/bash

# Execute user setup hook
if [ -f /usersetup.sh ]
then
    . /usersetup.sh
fi

# Execute the ctest script
exec ctest -S /entrypoint.cmake "$@"
