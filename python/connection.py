import hvac


class Connection:

    def __init__(self, vault_url: str, name: str, id: str, token: str):
        self.vault_url = vault_url
        self.name = name
        self.id = id
        self.token = token

    def resolve_credentials(self):
        vault_client = self.get_vault_client()
        result = vault_client.read(f'secret/data/{self.name}/{self.id}')
        try:
            return result['data']['data']
        except KeyError:
            raise RuntimeError(f'Invalid secret structure, failed resolving {self.name}')

    def get_vault_client(self):
        vault_client = hvac.Client(
            url=self.vault_url,
        )

        vault_client.token = self.token
        if not vault_client.is_authenticated():
            raise RuntimeError('Unable to authenticate to the Vault service')

        return vault_client
