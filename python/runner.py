import argparse
import json
import sys
import traceback
import runpy
import tempfile

from io import StringIO
from traceback import StackSummary
from typing import Dict

from context import Context
from exception import CodeExecutionError


def write_output(output: str, context: Context, error):
    output_struct = {
        "output": output,
        "error": str(error),
    }

    if context is not None:
        output_struct['context'] = context.map.toDict()

    sys.stdout.write(json.dumps(output_struct))


def decode_raw_input(raw_input_file) -> dict:
    with open(raw_input_file) as f:
        raw_input = f.read()

    # Flatten all the json values recursively,,,
    def flatten_hook(obj):
        for key, value in obj.items():
            if isinstance(value, str):
                try:
                    obj[key] = json.loads(value, object_hook=flatten_hook)
                except ValueError:
                    pass
        return obj

    decoded_input_json = json.loads(raw_input, object_hook=flatten_hook)
    return decoded_input_json



def execute_user_supplied_code(context: Context, code_to_be_executed: str):
    with tempfile.NamedTemporaryFile(mode='w') as f:
        f.write(code_to_be_executed)
        f.flush()

        runpy.run_path(f.name, init_globals=locals())


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
        # Note: All 'by value' list accesses are safe due to the python spec.
        error_line = StackSummary.extract(traceback.walk_tb(sys.exc_info()[2]))[-1].lineno
        raise CodeExecutionError(
            f'User provider code raised an exception: \n\r{str(e)}\n\r{type(e)}\n\rLine: {error_line}')
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
    except CodeExecutionError as e:
        write_output(output="", error=e, context=None)
        return
    except:
        write_output(output="", error=traceback.format_exc(), context=None)
        return

    write_output(output=output_buffer.getvalue(), error="", context=context)


if __name__ == '__main__':
    main()
