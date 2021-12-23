Feature:  Handling cleaning up remote files

  Scenario: all remote files are synced
   When the cloud has file "file1"
   And the cloud has file "file2"
   And the global config has file "file1"
   And the global config has file "file2"
   And the remote state data file does not exist
   Then the remote state data should be empty

  Scenario: file in cloud not in global config
   When the cloud has file "/dir1/file1"
   And the cloud has file "/dir1/file2"
   And the global config has file "/dir1/file1"
   And the remote state data file does not exist
   Then the remote state data should have file "/dir1/file2"

  Scenario: file in cloud under directory in global config
   When the cloud has file "/dir1/file1"
   And the global config has file "/dir1"
   And the remote state data file does not exist
   Then the remote state data should be empty

  Scenario: parent directory not in global config
   When the cloud has file "/dir1"
   And the cloud has file "/dir1/dir2"
   And the cloud has file "/dir1/dir2/file1"
   And the global config has file "/dir1/dir2"
   And the remote state data file does not exist
   Then the remote state data should be empty

  Scenario: different directory not in global config
   When the cloud has file "/dir1/dir3"
   And the cloud has file "/dir1/dir3/file1"
   And the global config has file "/dir1/dir2"
   And the remote state data file does not exist
   Then the remote state data should have file "/dir1/dir3"
   And the remote state data should have file "/dir1/dir3/file1"
