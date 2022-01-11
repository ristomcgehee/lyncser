import os
import subprocess
import tempfile
from typing import List, Tuple

config_dir_container = '/lyncser_config'
data_dir_container = '/lyncser_data'

class LyncserClient(object):
    def __init__(self, container_id):
        self.container_id = container_id
    
    def run_lyncser(self, args: List[str]):
        subprocess.run([
            'docker',
            'exec', 
            self.container_id, 
            'lyncser'] + args + ['--log-level=debug'],
        check=True)
    
    def write_data_file(self, filename: str, content: str):
        self._write_file(f'{data_dir_container}/{filename}', content)

    def write_config_file(self, filename: str, content: str):
        self._write_file(f'{config_dir_container}/{filename}', content)

    def get_data_file_content(self, filename: str) -> str:
        return self._get_file_content(f'{data_dir_container}/{filename}')

    def _write_file(self, filename: str, content: str):
        with tempfile.NamedTemporaryFile(mode='w+') as f:
            f.write(content)
            f.flush()
            subprocess.run([
                'docker',
                'cp',
                f.name,
                f'{self.container_id}:{filename}'],
            check=True)

    def _get_file_content(self, filename: str) -> str:
        result = subprocess.run([
            'docker',
            'exec',
            self.container_id,
            'cat',
            filename],
        check=True, capture_output=True)
        return result.stdout.decode('utf-8')


def set_global_config(files_to_sync: List[str], client: LyncserClient):
    files_str = '\n    - '.join([ os.path.join(data_dir_container, file) for file in files_to_sync ])
    global_config = f"""paths:
  all:
    - {files_str}
"""
    client.write_config_file('globalConfig.yaml', global_config)

def create_and_prep_clients(files_to_sync: List[str]) -> Tuple[LyncserClient, LyncserClient]:
    client1 = create_client()
    client2 = create_client()
    encryption_key = '166d8e96ae29d01dd155f840ac61657acfaa63bc24d15457183e9da03d33ef56'
    client1.write_config_file('encryption.key', encryption_key)
    client2.write_config_file('encryption.key', encryption_key)
    client1.run_lyncser(['deleteAllRemoteFiles', '-y'])

    set_global_config(files_to_sync, client1)
    set_global_config(files_to_sync, client2)

    return client1, client2

def create_client() -> LyncserClient:
    result = subprocess.run([
        'docker',
        'run',
        '-d', '-i',
        'lyncser-test'
    ], check=True, capture_output=True)
    container_id = result.stdout.decode('utf-8').strip()

    # Create symlink so lyncser can find the config directory
    subprocess.run(['docker', 'exec', container_id, 'bash', '-c', f'mkdir ~/.config && ln -s {config_dir_container} ~/.config/lyncser'])

    client = LyncserClient(container_id)
    creds = os.environ.get('LYNCSER_CREDENTIALS')
    token = os.environ.get('LYNCSER_TOKEN')
    client.write_config_file('credentials.json', creds)
    client.write_config_file('token.json', token)

    return client

