Feature: Syncing files at appropriate times

  Scenario: uploading file when modified locally
    When the file exists in the cloud
    And the cloud modified time is "7 am"
    And the file exists locally
    And the local modified time is "9 am"
    And the last cloud update was "8 am"
    Then the file should be uploaded to the cloud

  Scenario: uploading new file to cloud
    When the file does not exist in the cloud
    And the file exists locally
    And the local modified time is "7 am"
    And the last cloud update was "never"
    Then the file should be uploaded to the cloud

  Scenario: file doesn't exist anywhere
    When the file does not exist in the cloud
    And the file does not exist locally
    And the last cloud update was "never"
    Then nothing should happen

  Scenario: downloading new file from cloud
    When the file exists in the cloud
    And the file does not exist locally
    And the cloud modified time is "7 am"
    And the last cloud update was "never"
    Then the file should be downloaded from the cloud

  Scenario: downloading existing file from cloud
    When the file exists in the cloud
    And the file exists locally
    And the cloud modified time is "9 am"
    And the local modified time is "8 am"
    And the last cloud update was "8 am"
    Then the file should be downloaded from the cloud

  Scenario: downloading existing file from cloud when it hasn't been synced before
    When the file exists in the cloud
    And the file exists locally
    And the cloud modified time is "9 am"
    And the local modified time is "8 am"
    And the last cloud update was "never"
    Then the file should be downloaded from the cloud

  Scenario: do nothing when cloud modified time is recent
    When the file exists in the cloud
    And the cloud modified time is "8 am"
    And the file exists locally
    And the local modified time is "7 am"
    And the last cloud update was "8 am"
    Then nothing should happen

  Scenario: do nothing when last cloud update equals local modified time
    When the file exists in the cloud
    And the cloud modified time is "8 am"
    And the file exists locally
    And the local modified time is "9 am"
    And the last cloud update was "9 am"
    Then nothing should happen

  Scenario: do nothing when last cloud update is slightly after local modified time
    When the file exists in the cloud
    And the cloud modified time is "8 am"
    And the file exists locally
    And the local modified time is "9 am"
    And the last cloud update was "9:01 am"
    Then nothing should happen

  Scenario: upload file when last cloud update is slightly before local modified time
    When the file exists in the cloud
    And the cloud modified time is "7 am"
    And the file exists locally
    And the local modified time is "9:01 am"
    And the last cloud update was "9 am"
    Then the file should be uploaded to the cloud

  Scenario: mark file deleted when it exists in the cloud but not locally
    When the file exists in the cloud
     And the cloud modified time is "7 am"
    And the file does not exist locally
    And the last cloud update was "9 am"
    Then the file should be marked deleted locally

  Scenario: mark file deleted when it exists in the cloud but not locally
    When the file exists in the cloud
    And the cloud modified time is "7 am"
    And the file does not exist locally
    And the file was marked deleted locally
    Then nothing should happen

  Scenario: force download when the local modified time is after the cloud modified time
    When the file exists in the cloud
    And the cloud modified time is "7 am"
    And the file exists locally
    And the local modified time is "9 am"
    And force download is true
    Then the file should be downloaded from the cloud
