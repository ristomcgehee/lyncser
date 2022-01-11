import random

from .lyncser_client import create_and_prep_clients, set_global_config


def test_lot_of_stuff():
    file1 = f'test{random.randrange(0, 100000)}.txt'
    client1, client2 = create_and_prep_clients([file1])

    # Upload new file and download it on the other client
    file1_contents = 'test1'
    client1.write_data_file(file1, file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    assert client2.get_data_file_content(file1) == file1_contents

    # Modify the file on the first client and download it on the other client
    file1_contents = 'test2'
    client1.write_data_file(file1, file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    assert client2.get_data_file_content(file1) == file1_contents

    # Modify the file on the second client and download it on the first client
    file1_contents = 'test3'
    client2.write_data_file(file1, file1_contents)
    client2.run_lyncser(['sync'])
    client1.run_lyncser(['sync'])
    assert client2.get_data_file_content(file1) == file1_contents

    # Add a second file on the first client and make sure it gets downloaded on the other side
    while True:
        file2 = f'test{random.randrange(0, 100000)}.txt'
        if file1 != file2:
            break
    file2_contents = 'test4'
    client1.write_data_file(file2, file2_contents)
    set_global_config([file1, file2], client1)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    client2.run_lyncser(['sync']) # Run twice because it needs to pick up the change in the global config
    assert client2.get_data_file_content(file2) == file2_contents
