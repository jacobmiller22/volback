"""
e2e.py provides tests for e2e volback behavior
"""

import io
import boto3
import secrets
import botocore.client
import testcontainers.localstack
import pathlib
import pytest
import subprocess

PROJECT_ROOT_PATH = pathlib.Path(__file__).parent.parent.absolute()
SHELL_PATH = PROJECT_ROOT_PATH
BIN_PATH = PROJECT_ROOT_PATH.joinpath("bin/volback")
CMD_PATH = PROJECT_ROOT_PATH.joinpath("cmd/volback")
BUILD_CMD = f"go build -o {BIN_PATH} {CMD_PATH}"
E2E_ROOT_PATH = PROJECT_ROOT_PATH.joinpath("e2e")

LOCALSTACK_IMAGE = "localstack/localstack"

S3_BUCKET_NAME_1 = "bucket-a"
S3_BUCKET_NAME_2 = "bucket-b"

VOLBACK_DEFAULT_ENV = {
    "AWS_REGION": "us-east-1",
}


@pytest.fixture(scope="session", autouse=True)
def build():
    cp = subprocess.run(BUILD_CMD.split(" "))
    assert cp.returncode == 0


def test_binary_built():
    assert BIN_PATH.exists()


@pytest.fixture(scope="session")
def ls():
    with testcontainers.localstack.LocalStackContainer(LOCALSTACK_IMAGE) as ls:
        ls.start()
        yield ls


@pytest.fixture(scope="session")
def lsendpointurl(ls):
    return ls.get_url()


@pytest.fixture(scope="session")
def s3client(lsendpointurl):
    client = boto3.resource(
        "s3",
        region_name="us-east-1",
        endpoint_url=lsendpointurl,
        aws_access_key_id="test",
        aws_secret_access_key="test",
        config=botocore.client.Config(s3={"addressing_style": "path"}),
    )

    client.Bucket(S3_BUCKET_NAME_1).create()
    client.Bucket(S3_BUCKET_NAME_2).create()
    return client


@pytest.fixture(scope="function")
def cleanup_testdata():
    yield
    # os.remove("./testdata/generated")


def encryption_key(size: int):
    """Randomly generate an encryption key for each test."""

    return secrets.token_hex(32)[:size]


def test_e2e_fs2fs(cleanup_testdata):
    """
    This test will backup a file, and restore the file and confirm nothing was lost.

    Source: fs
    Destination: fs
    """

    k = encryption_key(17)

    original_path = E2E_ROOT_PATH.joinpath("./testdata/lorem.pt")
    encrypted_path = E2E_ROOT_PATH.joinpath("./testdata/generated/lorem-0.pt.ct")
    restored_path = E2E_ROOT_PATH.joinpath("./testdata/generated/lorem-0.pt.ct.pt")

    cp = subprocess.run(
        [
            BIN_PATH,
            "-f",
            E2E_ROOT_PATH.joinpath("./testdata/backup_fs2fs.json").as_posix(),
            "--enc.key",
            k,
        ],
        env=VOLBACK_DEFAULT_ENV,
    )
    assert cp.returncode == 0

    cp = subprocess.run(
        [
            BIN_PATH,
            "-f",
            E2E_ROOT_PATH.joinpath("./testdata/backup_fs2fs_restore.json").as_posix(),
            "--enc.key",
            k,
        ],
        env=VOLBACK_DEFAULT_ENV,
    )
    assert cp.returncode == 0

    original_data = open(original_path, "rb").read()
    encrypted_data = open(encrypted_path, "rb").read()
    restored_data = open(restored_path, "rb").read()

    assert original_data != encrypted_data
    assert original_data == restored_data


def test_e2e_fs2s3(cleanup_testdata, s3client, lsendpointurl):
    """
    This test will backup a file, and restore the file and confirm nothing was lost.

    Source: fs
    Destination: s3
    """

    k = encryption_key(23)

    original_path = E2E_ROOT_PATH.joinpath("./testdata/lorem.pt")
    encrypted_path = "testdata/generated/lorem-1.pt.ct"
    restored_path = E2E_ROOT_PATH.joinpath("./testdata/generated/lorem-1.pt.ct.pt")

    cp = subprocess.run(
        [
            BIN_PATH,
            "-f",
            E2E_ROOT_PATH.joinpath("./testdata/backup_fs2s3.json").as_posix(),
            "--src.path",
            original_path,
            "--enc.key",
            k,
            "--dst.path",
            encrypted_path,
            "--dst.s3-endpoint",
            lsendpointurl,
            "--dst.s3-bucket",
            S3_BUCKET_NAME_1,
        ],
        env={**VOLBACK_DEFAULT_ENV, "S3_FORCE_PATH_STYLE": "true"},
    )
    assert cp.returncode == 0

    cp = subprocess.run(
        [
            BIN_PATH,
            "-f",
            E2E_ROOT_PATH.joinpath("./testdata/backup_fs2s3_restore.json").as_posix(),
            "--src.path",
            encrypted_path,
            "--src.s3-endpoint",
            lsendpointurl,
            "--src.s3-bucket",
            S3_BUCKET_NAME_1,
            "--enc.key",
            k,
            "--dst.path",
            restored_path,
        ],
        env={**VOLBACK_DEFAULT_ENV, "S3_FORCE_PATH_STYLE": "true"},
    )
    assert cp.returncode == 0

    original_data = open(original_path, "rb").read()
    encrypted_data = io.BytesIO()
    s3client.Bucket(S3_BUCKET_NAME_1).download_fileobj(encrypted_path, encrypted_data)
    restored_data = open(restored_path, "rb").read()

    assert original_data != encrypted_data
    assert original_data == restored_data


def test_fs2fs_no_config(cleanup_testdata):
    """
    This test will backup a file, and restore the file and confirm nothing was lost. All while using no config file.

    Source: fs
    Destination: fs
    """

    k = encryption_key(17)

    original_path = E2E_ROOT_PATH.joinpath("./testdata/lorem.pt")
    encrypted_path = E2E_ROOT_PATH.joinpath("./testdata/generated/lorem-0.pt.ct")
    restored_path = E2E_ROOT_PATH.joinpath("./testdata/generated/lorem-0.pt.ct.pt")

    # {
    # 	"source": {
    # 		"kind": "fs",
    # 		"path": "testdata/lorem.pt"
    # 	},
    # 	"restore": false,
    # 	"encryption": {
    # 		"key": "temp size 16 key"
    # 	},
    # 	"destination": {
    # 		"kind": "fs",
    # 		"path": "testdata/generated/lorem-0.pt.ct"
    # 	}
    # }

    cp = subprocess.run(
        [
            BIN_PATH,
            "-src.kind",
            "fs",
            "-src.path",
            original_path.absolute().as_posix(),
            "--restore=false",
            "--enc.key",
            k,
            "-dst.kind",
            "fs",
            "-dst.path",
            encrypted_path.absolute().as_posix(),
        ]
    )
    assert cp.returncode == 0

    cp = subprocess.run(
        [
            BIN_PATH,
            "-src.kind",
            "fs",
            "-src.path",
            encrypted_path.absolute().as_posix(),
            "--restore=true",
            "-dst.kind",
            "fs",
            "-dst.path",
            restored_path.absolute().as_posix(),
            "--enc.key",
            k,
        ]
    )
    assert cp.returncode == 0

    original_data = open(original_path, "rb").read()
    encrypted_data = open(encrypted_path, "rb").read()
    restored_data = open(restored_path, "rb").read()

    assert original_data != encrypted_data
    assert original_data == restored_data
