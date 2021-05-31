class Context:
    _KEY_SEPARATOR = '.'

    def __init__(self, internal_dict: dict):
        self.internal_dict: dict = internal_dict

    def __getitem__(self, item: str):
        return self.__resolve_inner_key(item)

    def __setitem__(self, key, value):
        key_parts = key.split(self._KEY_SEPARATOR)
        last_key = key_parts.pop()

        key_off_by_one = "".join(key_parts)
        item = self.__resolve_inner_key(key_off_by_one)
        item[last_key] = value

    def __resolve_inner_key(self, key):
        key_parts = key.split(self._KEY_SEPARATOR)

        current_item = self.internal_dict
        for key_part in key_parts:
            if type(current_item) != dict:
                raise ValueError(f'Key {key} does not exist')

            current_item = current_item.__getitem__(key_part)

        return current_item
