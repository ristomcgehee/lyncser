Feature: file store operations

  Scenario: uploading file when modified locally
    When the file exists in the cloud
    And the cloud modified time is "7 am"
    And the file exists locally
    And the local modified time is "9 am"
    And the last cloud update was "8 am"
    Then the file should be updated to the cloud
