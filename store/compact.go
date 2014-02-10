package store

import (
	"fmt"
	"github.com/cloudfoundry/storeadapter"
	"regexp"
	"strconv"
	"strings"
)

func (store *RealStore) Compact() error {
	err := store.deleteOldSchemaVersionsAndUnversionedData()
	if err != nil {
		return err
	}

	err = store.deleteEmptyDirectories()
	if err != nil {
		return err
	}
	return nil
}

func (store *RealStore) deleteOldSchemaVersionsAndUnversionedData() error {
	everything, err := store.adapter.ListRecursively("/hm")
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`^/hm/v(\d+)$`)

	keysToDelete := []string{}
	for _, childNode := range everything.ChildNodes {
		if strings.HasPrefix(childNode.Key, "/hm/locks") {
			continue
		}
		matches := re.FindStringSubmatch(childNode.Key)
		if len(matches) == 2 {
			schemaVersion, err := strconv.Atoi(matches[1])
			if err != nil {
				keysToDelete = append(keysToDelete, childNode.Key)
				continue
			}
			if schemaVersion < store.config.StoreSchemaVersion {
				keysToDelete = append(keysToDelete, childNode.Key)
			}
		} else {
			keysToDelete = append(keysToDelete, childNode.Key)
		}
	}

	return store.adapter.Delete(keysToDelete...)
}

func (store *RealStore) deleteEmptyDirectories() error {
	node, err := store.adapter.ListRecursively(store.SchemaRoot() + "/")
	if err != nil {
		store.logger.Error(fmt.Sprintf("Failed to recursively fetch %s/", store.SchemaRoot()), err)
		return err
	}

	store.deleteEmptyDirectoriesUnder(node)
	return nil
}

func (store *RealStore) deleteEmptyDirectoriesUnder(node storeadapter.StoreNode) bool {
	if node.Dir {
		if len(node.ChildNodes) == 0 {
			// ignoring errors -- best effort!
			store.logger.Info("Deleting Key", map[string]string{"Key": node.Key})
			store.adapter.Delete(node.Key)
			return true
		} else {
			deletedAll := true

			for _, child := range node.ChildNodes {
				deletedAll = store.deleteEmptyDirectoriesUnder(child) && deletedAll
			}

			if deletedAll {
				// ignoring errors -- best effort!
				store.logger.Info("Deleting Key", map[string]string{"Key": node.Key})
				store.adapter.Delete(node.Key)
				return true
			}
		}
	}

	return false
}
