from os import mkdir
import os
import random
import subprocess
import tempfile
from typing import List, Tuple

config_dir_container = '/lyncser_config'
data_dir_container = '/lyncser_data'


class LyncserClient(object):
    def __init__(self, config_dir_host, data_dir_host, container_id):
        self.config_dir_host = config_dir_host
        self.data_dir_host = data_dir_host
        self.container_id = container_id
    
    def run_lyncser(self, args: List[str]):
        subprocess.run([
            'docker',
            'exec', 
            self.container_id, 
            'lyncser'] + args + ['--log-level=debug'],
        check=True)
    
    def write_data_file(self, filename: str, content: str):
        data_file_path = os.path.join(self.data_dir_host, filename)
        with open(data_file_path, 'w') as f:
            f.write(content)

    def get_data_file_content(self, filename: str) -> str:
        data_file_path = os.path.join(self.data_dir_host, filename)
        with open(data_file_path, 'r') as f:
            return f.read()
    
    def write_config_file(self, filename: str, content: str):
        config_file_path = os.path.join(self.config_dir_host, filename)
        with open(config_file_path, 'w') as f:
            f.write(content)
    
    def get_config_file_content(self, filename: str) -> str:
        config_file_path = os.path.join(self.config_dir_host, filename)
        with open(config_file_path, 'r') as f:
            return f.read()


def create_and_prep_clients(files_to_sync: List[str]) -> Tuple[LyncserClient, LyncserClient]:
    client1 = create_client()
    client2 = create_client()
    encryption_key = '166d8e96ae29d01dd155f840ac61657acfaa63bc24d15457183e9da03d33ef56'
    client1.write_config_file('encryption.key', encryption_key)
    client2.write_config_file('encryption.key', encryption_key)
    client1.run_lyncser(['deleteAllRemoteFiles', '-y'])

    files_str = '\n    - '.join([ os.path.join(data_dir_container, file) for file in files_to_sync ])
    global_config = f"""paths:
  all:
    - "{files_str}"
"""
    client1.write_config_file('globalConfig.yaml', global_config)
    client2.write_config_file('globalConfig.yaml', global_config)

    return client1, client2

def test_upload_download():
    file1 = f'test{random.randrange(0, 100000)}.txt'
    client1, client2 = create_and_prep_clients([file1])
    file1_contents = 'test1'
    client1.write_data_file(file1, file1_contents)
    client1.run_lyncser(['sync'])
    client2.run_lyncser(['sync'])

    assert client2.get_data_file_content(file1) == file1_contents

def create_client() -> LyncserClient:
    lyncser_dir = tempfile.mkdtemp(prefix='lyncser_test_')
    config_dir_host = os.path.join(lyncser_dir, 'config')
    mkdir(config_dir_host)
    data_dir_host = os.path.join(lyncser_dir, 'data')
    mkdir(data_dir_host)
    result = subprocess.run([
        'docker',
        'run',
        '-v', f'{config_dir_host}:{config_dir_container}',
        '-v', f'{data_dir_host}:/{data_dir_container}',
        '-d', '-i',
        'lyncser-test'
    ], check=True, capture_output=True)
    container_id = result.stdout.decode('utf-8').strip()

    # Create symlink so lyncser can find the config directory
    subprocess.run(['docker', 'exec', container_id, 'bash', '-c', f'mkdir ~/.config && ln -s {config_dir_container} ~/.config/lyncser'])

    client = LyncserClient(config_dir_host, data_dir_host, container_id)
    creds = os.environ.get('LYNCSER_CREDENTIALS')
    token = os.environ.get('LYNCSER_TOKEN')
    client.write_config_file('credentials.json', creds)
    client.write_config_file('token.json', token)

    return client







