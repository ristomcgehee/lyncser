import random

from .lyncser_client import create_and_prep_clients


def test_upload_download():
    file1 = f'test{random.randrange(0, 100000)}.txt'
    client1, client2 = create_and_prep_clients([file1])
    file1_contents = 'test1'
    client1.write_data_file(file1, file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])

    assert client2.get_data_file_content(file1) == file1_contents
