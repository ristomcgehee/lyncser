import os
import subprocess
import tempfile
from typing import Dict, List, Tuple

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

    
    def set_global_config(self, files_to_sync: Dict[str, List[str]]) -> None:
        files_str = '\n    - '.join([ os.path.join(data_dir_container, file) for file in files_to_sync ])
        global_config = "paths:\n"
        for tag, files in files_to_sync.items():
            files_str = '\n    - '.join([ os.path.join(data_dir_container, file) for file in files ])
            global_config += f"  {tag}:\n    - {files_str}\n"
        self.write_config_file('globalConfig.yaml', global_config)
    
    def set_local_tags(self, tags: List[str]) -> None:
        self.write_config_file('localConfig.yaml', 'tags:\n  - ' + '\n  - '.join(tags) + '\n')
    
    def write_data_file(self, filename: str, content: str) -> None:
        self._write_file(f'{data_dir_container}/{filename}', content)

    def write_config_file(self, filename: str, content: str) -> None:
        self._write_file(f'{config_dir_container}/{filename}', content)
    
    def create_data_dir(self, dir_name: str) -> None:
        subprocess.run([
            'docker',
            'exec',
            self.container_id,
            'mkdir',
            f'{data_dir_container}/{dir_name}'],
        check=True)
    
    def data_file_exists(self, filename: str) -> bool:
        result = subprocess.run([
            'docker',
            'exec',
            self.container_id,
            'test',
            '-f',
            f'{data_dir_container}/{filename}'
        ])
        return result.returncode == 0

    def get_data_file_content(self, filename: str) -> str:
        return self._get_file_content(f'{data_dir_container}/{filename}')
    
    def delete_data_file(self, filename: str) -> None:
        subprocess.run([
            'docker',
            'exec',
            self.container_id,
            'rm',
            f'{data_dir_container}/{filename}'],
        check=True)

    def _write_file(self, filename: str, content: str) -> None:
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


def create_and_prep_clients(files_to_sync: Dict[str, List[str]]) -> Tuple[LyncserClient, LyncserClient, LyncserClient]:
    encryption_key = '166d8e96ae29d01dd155f840ac61657acfaa63bc24d15457183e9da03d33ef56'
    clients_to_create = 3
    clients = ()
    for _ in range(clients_to_create):
        client = create_client()
        client.write_config_file('encryption.key', encryption_key)
        client.set_global_config(files_to_sync)
        clients += (client,)

    clients[0].run_lyncser(['deleteAllRemoteFiles', '-y'])
    return clients

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

