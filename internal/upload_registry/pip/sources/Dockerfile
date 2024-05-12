FROM python:latest

WORKDIR /project

COPY setup.py setup.py

COPY .pypirc /root/.pypirc

RUN pip install twine sdist wheel

RUN python setup.py sdist bdist_wheel

CMD [ "twine", "upload", "dist/*.tar.gz", "--verbose"]
