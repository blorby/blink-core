import argparse
import base64
import json
import sys

from io import StringIO

from typing import Dict

from context import Context
from connection import Connection


class CodeExecutionError(Exception):
    pass


def write_output(output: str, context: Context, error):
    output_struct = {
        "output": output,
        "error": str(error),
    }

    if context is not None:
        output_struct['context'] = context.internal_dict

    sys.stdout.write(json.dumps(output_struct))


def decode_raw_input(raw_input) -> dict:
    raw_input_json = base64.b64decode(raw_input)
    decoded_input_json = json.loads(raw_input_json)

    base64_encoded_python_code = decoded_input_json['code']
    decoded_python_code = base64.b64decode(base64_encoded_python_code)

    decoded_input_json['code'] = decoded_python_code
    return decoded_input_json


def build_connection_instances(raw_connections: dict):
    concrete_connections = {}
    for name, connection in raw_connections.items():
        concrete_connections[name] = Connection(name=name,
                                                id=connection['Id'],
                                                token=connection['Token'],
                                                vault_url=connection['VaultUrl'])

    return concrete_connections


def execute_user_supplied_code(context: Context, connections: Dict[str, Connection], code_to_be_executed: str):
    exec(code_to_be_executed)


def entry_point(raw_input):
    decoded_input = decode_raw_input(raw_input)

    context = Context(decoded_input['context'])
    code_to_be_executed = decoded_input['code']
    connections = build_connection_instances(decoded_input['connections'])

    try:
        output_buffer = StringIO()
        sys.stdout = output_buffer

        execute_user_supplied_code(context=context, connections=connections, code_to_be_executed=code_to_be_executed)
    except Exception as e:
        raise CodeExecutionError(f'User provider code raised an exception: {str(e)}\n{type(e)}')
    finally:
        sys.stdout = sys.__stdout__

    return output_buffer, context


def main():
    parser = argparse.ArgumentParser(description="Python action runner process wrapper")
    parser.add_argument(
        "--input", required=True, help="The raw marshaled input json struct"
    )

    arguments = parser.parse_args()

    try:
        output_buffer, context = entry_point(arguments.input)
    except Exception as e:
        write_output(output="", error=e, context=None)
        return

    write_output(output=output_buffer.getvalue(), error="", context=context)


if __name__ == '__main__':
    main()
