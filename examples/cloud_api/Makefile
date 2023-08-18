pytest: clean_cache
	docker build -t cloud-lt-examples .
	docker run -v $(PWD):$(PWD) -w $(PWD) cloud-lt-examples pytest

shell:
	docker build -t cloud-lt-examples .
	docker run -it -v $(PWD):$(PWD) -w $(PWD) cloud-lt-examples bash

clean_cache:
	rm -rf __pycache__
	rm -rf .pytest_cache