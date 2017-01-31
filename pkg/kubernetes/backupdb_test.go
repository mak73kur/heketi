//
// Copyright (c) 2017 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//

package kubernetes

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"

	"github.com/heketi/tests"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
	fakeclientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5/fake"
	"k8s.io/kubernetes/pkg/client/restclient"
)

func TestBackupToKubeSecretFailedClusterConfig(t *testing.T) {
	tmpfile := tests.Tempfile()
	defer os.Remove(tmpfile)

	// Create a db
	db, err := bolt.Open(tmpfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	tests.Assert(t, err == nil)
	defer db.Close()

	incluster_count := 0
	defer tests.Patch(&inClusterConfig, func() (*restclient.Config, error) {
		incluster_count++
		return nil, fmt.Errorf("TEST")
	}).Restore()

	config_count := 0
	defer tests.Patch(&newForConfig, func(c *restclient.Config) (clientset.Interface, error) {
		config_count++
		return nil, nil
	}).Restore()

	ns := "default"
	ns_count := 0
	defer tests.Patch(&getNamespace, func() (string, error) {
		ns_count++
		return ns, nil
	}).Restore()

	// Try to backup
	err = KubeBackupDbToSecret(db)
	tests.Assert(t, incluster_count == 1)
	tests.Assert(t, config_count == 0)
	tests.Assert(t, ns_count == 0)
	tests.Assert(t, err != nil)
}

func TestBackupToKubeSecretFailedNewConfig(t *testing.T) {
	tmpfile := tests.Tempfile()
	defer os.Remove(tmpfile)

	// Create a db
	db, err := bolt.Open(tmpfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	tests.Assert(t, err == nil)
	defer db.Close()

	incluster_count := 0
	defer tests.Patch(&inClusterConfig, func() (*restclient.Config, error) {
		incluster_count++
		return nil, nil
	}).Restore()

	config_count := 0
	defer tests.Patch(&newForConfig, func(c *restclient.Config) (clientset.Interface, error) {
		config_count++
		return nil, fmt.Errorf("TEST")
	}).Restore()

	ns := "default"
	ns_count := 0
	defer tests.Patch(&getNamespace, func() (string, error) {
		ns_count++
		return ns, nil
	}).Restore()

	// Try to backup
	err = KubeBackupDbToSecret(db)
	tests.Assert(t, incluster_count == 1)
	tests.Assert(t, config_count == 1)
	tests.Assert(t, ns_count == 0)
	tests.Assert(t, err != nil)
}

func TestBackupToKubeSecretFailedNamespace(t *testing.T) {
	tmpfile := tests.Tempfile()
	defer os.Remove(tmpfile)

	// Create a db
	db, err := bolt.Open(tmpfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	tests.Assert(t, err == nil)
	defer db.Close()

	incluster_count := 0
	defer tests.Patch(&inClusterConfig, func() (*restclient.Config, error) {
		incluster_count++
		return nil, nil
	}).Restore()

	config_count := 0
	defer tests.Patch(&newForConfig, func(c *restclient.Config) (clientset.Interface, error) {
		config_count++
		return nil, nil
	}).Restore()

	ns_count := 0
	defer tests.Patch(&getNamespace, func() (string, error) {
		ns_count++
		return "", fmt.Errorf("TEST")
	}).Restore()

	// Try to backup
	err = KubeBackupDbToSecret(db)
	tests.Assert(t, incluster_count == 1)
	tests.Assert(t, config_count == 1)
	tests.Assert(t, ns_count == 1)
	tests.Assert(t, err != nil)
}

func TestBackupToKubeSecretGoodBackup(t *testing.T) {
	tmpfile := tests.Tempfile()
	defer os.Remove(tmpfile)

	// Create a db
	db, err := bolt.Open(tmpfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	tests.Assert(t, err == nil)
	defer db.Close()

	incluster_count := 0
	defer tests.Patch(&inClusterConfig, func() (*restclient.Config, error) {
		incluster_count++
		return nil, nil
	}).Restore()

	config_count := 0
	defer tests.Patch(&newForConfig, func(c *restclient.Config) (clientset.Interface, error) {
		config_count++
		return fakeclientset.NewSimpleClientset(), nil
	}).Restore()

	ns := "default"
	ns_count := 0
	defer tests.Patch(&getNamespace, func() (string, error) {
		ns_count++
		return ns, nil
	}).Restore()

	err = KubeBackupDbToSecret(db)
	tests.Assert(t, incluster_count == 1)
	tests.Assert(t, config_count == 1)
	tests.Assert(t, ns_count == 1)
	tests.Assert(t, err == nil)
}

func TestBackupToKubeSecretVerifyBackup(t *testing.T) {
	tmpfile := tests.Tempfile()
	defer os.Remove(tmpfile)

	// Create a db
	db, err := bolt.Open(tmpfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	tests.Assert(t, err == nil)

	incluster_count := 0
	defer tests.Patch(&inClusterConfig, func() (*restclient.Config, error) {
		incluster_count++
		return nil, nil
	}).Restore()

	config_count := 0
	fakeclient := fakeclientset.NewSimpleClientset()
	defer tests.Patch(&newForConfig, func(c *restclient.Config) (clientset.Interface, error) {
		config_count++
		return fakeclient, nil
	}).Restore()

	ns := "default"
	ns_count := 0
	defer tests.Patch(&getNamespace, func() (string, error) {
		ns_count++
		return ns, nil
	}).Restore()

	// Add some content to the db
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("bucket"))
		tests.Assert(t, err == nil)

		err = bucket.Put([]byte("key1"), []byte("value1"))
		tests.Assert(t, err == nil)

		return nil
	})
	tests.Assert(t, err == nil)

	// Save to a secret
	err = KubeBackupDbToSecret(db)
	tests.Assert(t, incluster_count == 1)
	tests.Assert(t, config_count == 1)
	tests.Assert(t, ns_count == 1)
	tests.Assert(t, err == nil)

	// Get the secret
	secret, err := fakeclient.CoreV1().Secrets(ns).Get("heketi-db-backup")
	tests.Assert(t, err == nil)

	// Verify
	newdb := tests.Tempfile()
	defer os.Remove(newdb)
	err = ioutil.WriteFile(newdb, secret.Data["heketi.db"], 0644)
	tests.Assert(t, err == nil)

	// Load new app with backup
	db.Close()
	db, err = bolt.Open(tmpfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	tests.Assert(t, err == nil)
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("bucket"))
		tests.Assert(t, bucket != nil)

		val := bucket.Get([]byte("key1"))
		tests.Assert(t, val != nil)
		tests.Assert(t, string(val) == "value1")

		return nil
	})
	tests.Assert(t, err == nil)
}
