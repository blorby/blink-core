from dotmap import DotMap


class Context:

    def __init__(self, internal_dict: dict):
        object.__setattr__(self, 'map', DotMap(internal_dict))
        self.map.__class__.__name__ = '' # Override the stupid string representation

    def __getattr__(self, item):

        try:
            return object.__getattribute__(self, item)
        except:
            pass

        return object.__getattribute__(self, 'map').__getattr__(item)

    def __setattr__(self, key, value):
        self.map.__setattr__(key, value)

    def __str__(self):
        return str(self.map.toDict())
