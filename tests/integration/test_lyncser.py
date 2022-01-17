import random

from .lyncser_client import create_and_prep_clients, LyncserClient, cleanup_clients


def test_lot_of_stuff():
    file1 = f'test{random.randrange(0, 100000)}.txt'
    files_to_sync = { 'all': [file1]}
    client1, client2, client3 = create_and_prep_clients(files_to_sync)

    # Upload new file and download it on client2
    file1_contents = 'test1'
    client1.write_data_file(file1, file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    assert client2.get_data_file_content(file1) == file1_contents

    # Modify the file on client1 and download it on client2
    file1_contents = 'test1_modified'
    client1.write_data_file(file1, file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    assert client2.get_data_file_content(file1) == file1_contents

    # Modify the file on client2 and download it on client1
    file1_contents = 't'
    client2.write_data_file(file1, file1_contents)
    client2.run_lyncser(['sync'])
    client1.run_lyncser(['sync'])
    assert client2.get_data_file_content(file1) == file1_contents

    # Add file2 on client1 and make sure it gets downloaded on client2
    while True:
        file2 = f'test{random.randrange(0, 100000)}.txt'
        if file1 != file2:
            break
    file2_contents = 'test4'
    client1.write_data_file(file2, file2_contents)
    files_to_sync = { 'all': [file1, file2]}
    client1.set_global_config(files_to_sync)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    client2.run_lyncser(['sync']) # Run again to pick up the change in the global config
    assert client2.get_data_file_content(file2) == file2_contents

    # Delete file2 on the client1 and make sure it doesn't gets deleted on client2
    client1.delete_data_file(file2)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    assert client2.get_data_file_content(file2) == file2_contents
    assert not client1.data_file_exists(file2)

    # Remove file2 from being synced on client2 and make sure it's no longer synced
    files_to_sync = { 'all': [file1], 'client1and3': [file2]}
    client1.set_global_config(files_to_sync)
    client1.set_local_tags(['all', 'client1and3'])
    client3.set_local_tags(['all', 'client1and3'])
    # Propagate the global config to all clients
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    client3.run_lyncser(['sync'])
    file2_contents_client2 = file2_contents
    file2_contents_client1 = 'test5'
    client1.write_data_file(file2, file2_contents_client1)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    client3.run_lyncser(['sync'])
    assert client2.get_data_file_content(file2) == file2_contents_client2
    assert client3.get_data_file_content(file2) == file2_contents_client1
    file2_contents_client2 = 'test6'
    client2.write_data_file(file2, file2_contents_client2)
    client2.run_lyncser(['sync'])
    client1.run_lyncser(['sync'])
    client3.run_lyncser(['sync'])
    assert client1.get_data_file_content(file2) == file2_contents_client1
    assert client3.get_data_file_content(file2) == file2_contents_client1

    _test_directory_stuff(client1, client2)

    # Only run cleanup if tests passed so we can examine the docker containers after failures.
    cleanup_clients([client1, client2, client3])

def _test_directory_stuff(client1: LyncserClient, client2: LyncserClient) -> None:
    # Sync a directory
    dir1 = 'test_dir'
    client1.create_data_dir(dir1)
    files_to_sync = { 'all': [dir1]}
    client1.set_global_config(files_to_sync)
    dir1_file1 = f'{dir1}/test1.txt'
    dir1_file1_contents = 'test1'
    client1.write_data_file(dir1_file1, dir1_file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])
    client2.run_lyncser(['sync']) # Run again to pick up the change in the global config
    assert client2.get_data_file_content(dir1_file1) == dir1_file1_contents

    # Delete file in directory on client2 and make sure it's not deleted on client1
    client2.delete_data_file(dir1_file1)
    client2.run_lyncser(['sync'])
    client1.run_lyncser(['sync'])
    assert client1.get_data_file_content(dir1_file1) == dir1_file1_contents
