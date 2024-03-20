REF=dckr.woland.xyz/zte-5g
CTX=creep

.PHONY: image
.PHONY: deploy
.PHONY: logs

image:
	docker build -t $(REF) .
	docker push $(REF)

deploy: image
	docker -c $(CTX) compose up -d --pull always

logs:
	docker -c $(CTX) compose logs -f -n100
