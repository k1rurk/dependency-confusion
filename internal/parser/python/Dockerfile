FROM python:latest


RUN pip3 install twine sdist wheel
RUN python3 setup.py sdist wheel
RUN twine upload dist/* -u nuna789 -p 123
