import os

import pytest
import yandexcloud
from dotenv import load_dotenv

from common import ENV_VARS

@pytest.fixture
def sdk():
    load_dotenv()
    token = ENV_VARS.token()
    assert token, 'Token should be specified'
    sdk = yandexcloud.SDK(iam_token=token)
    return sdk