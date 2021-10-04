import argparse
import base64
import json
import sys

from io import StringIO

from typing import Dict

from context import Context
from connection import Connection
from exception import CodeExecutionError


def write_output(output: str, context: Context, error):
    output_struct = {
        "output": output,
        "error": str(error),
    }

    if context is not None:
        output_struct['context'] = context.internal_dict

    sys.stdout.write(json.dumps(output_struct))


def decode_raw_input(raw_input_file) -> dict:
    raw_input = ""
    with open(raw_input_file) as f:
        raw_input = f.read()

    decoded_input_json = json.loads(raw_input)
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


def entry_point(raw_input_file):
    decoded_input = decode_raw_input(raw_input_file)

    context = Context(decoded_input['context'])
    code_to_be_executed = decoded_input['code']
    connections = build_connection_instances(decoded_input['connections'])

    try:
        output_buffer = StringIO()
        sys.stdout = output_buffer

        execute_user_supplied_code(context=context, connections=connections, code_to_be_executed=code_to_be_executed)
    except Exception as e:
        raise CodeExecutionError(f'User provided code raised an exception: {str(e)}\n{type(e)}')
    finally:
        sys.stdout = sys.__stdout__

    return output_buffer, context


def main():
    parser = argparse.ArgumentParser(description="Python action runner process wrapper")
    parser.add_argument(
        "--input", required=True, help="File location containing the raw marshaled input json struct"
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
