import json

from exception import ContextStructureError


class Context:
    _KEY_SEPARATOR = '.'
    _VARIABLES_PREFIX = 'variables'

    def __init__(self, internal_dict: dict):
        self.internal_dict: dict = internal_dict

    def __getitem__(self, item: str):
        inner_item = self.__resolve_inner_key(key=item)
        if type(inner_item) == dict:
            return json.dumps(obj=inner_item, indent=4)
        return inner_item

    def __setitem__(self, key, value):
        key_parts = key.split(self._KEY_SEPARATOR)
        last_key = key_parts.pop()

        if len(key_parts) == 0:
            raise KeyError(f'Key {key} does not exist')

        key_off_by_one = self._KEY_SEPARATOR.join(key_parts)
        item = self.__resolve_inner_key(key=key_off_by_one, create_keys=True)
        if type(item) != dict:
            raise ContextStructureError(f'Key {key} already exists and not a tree')

        item[last_key] = value

    def __resolve_inner_key(self, key, create_keys: bool = False):
        key_parts = key.split(self._KEY_SEPARATOR)

        current_item = self.internal_dict
        for key_part in key_parts:
            if type(current_item) != dict:
                if not create_keys:
                    raise KeyError(f'Key {key} does not exist')

                current_item = dict()

            if not current_item.__contains__(key_part):
                if create_keys:
                    current_item[key_part] = dict()
                else:
                    raise KeyError(f'Key {key} does not exist')

            current_item = current_item.__getitem__(key_part)

        return current_item

    def get(self, key):
        return self.__getitem__(key)

    def set(self, key, value):
        path = self._validate_prefix(key)
        key = self._KEY_SEPARATOR.join(path)
        self.__setitem__(key, value)

    def delete(self, key):
        path = self._validate_prefix(key)
        key_to_delete = path.pop(len(path) - 1)
        key = self._KEY_SEPARATOR.join(path)
        parent_dict = self.__resolve_inner_key(key)
        parent_dict.pop(key_to_delete, None)

    def _validate_prefix(self, key):
        path = key.split(self._KEY_SEPARATOR)

        if path[0] != self._VARIABLES_PREFIX:
            raise KeyError(f'Key {key} is invalid')

        return path
