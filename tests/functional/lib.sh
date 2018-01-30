#!/bin/bash

fail() {
    echo "==> ERROR: $*"
    exit 1
}

println() {
    echo "==> $1"
}

_sudo() {
    if [ ${UID} = 0 ] ; then
        "${@}"
    else
        sudo -E "${@}"
    fi
}

HEKETI_PID=
start_heketi() {
    ( cd "$HEKETI_SERVER_BUILD_DIR" && make && cp heketi "$HEKETI_SERVER" )
    if [ $? -ne 0 ] ; then
        fail "Unable to build Heketi"
    fi

    # Start server
    rm -f heketi.db > /dev/null 2>&1
    $HEKETI_SERVER --config=config/heketi.json &
    HEKETI_PID=$!
    sleep 2
}

start_vagrant() {
    cd vagrant || fail "Unable to 'cd vagrant'."
    _sudo ./up.sh || fail "unable to start vagrant virtual machines"
    cd ..
}

teardown_vagrant() {
    cd vagrant || fail "Unable to 'cd vagrant'."
    _sudo vagrant destroy -f
    cd ..
}

run_tests() {
    cd tests || fail "Unable to 'cd tests'."
    go test -timeout=1h -tags functional -v
    gotest_result=$?
    cd ..
}

force_cleanup_libvirt_disks() {
    # Sometimes disks are not deleted
    for i in $(_sudo virsh vol-list default | grep '\.disk' | awk '{print $1}') ; do
        _sudo virsh vol-delete --pool default "${i}" || fail "Unable to delete disk $i"
    done
}

teardown() {
    if [[ "$HEKETI_TEST_VAGRANT" != "no" ]]
    then
        teardown_vagrant
        force_cleanup_libvirt_disks
    fi
    rm -f heketi.db > /dev/null 2>&1
}

functional_tests() {
    if [[ "$HEKETI_TEST_VAGRANT" != "no" ]]
    then
        start_vagrant
    fi
    start_heketi

    run_tests

    kill $HEKETI_PID
    if [[ "$HEKETI_TEST_CLEANUP" != "no" ]]
    then
        teardown
    fi

    exit $gotest_result
}

