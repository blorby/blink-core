from exception import ContextStructureError


class Context:
    _KEY_SEPARATOR = '.'

    def __init__(self, internal_dict: dict):
        self.internal_dict: dict = internal_dict

    def __getitem__(self, item: str):
        return self.__resolve_inner_key(key=item)

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
