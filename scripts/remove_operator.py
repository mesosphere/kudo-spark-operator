#!/usr/bin/env python3

import subprocess
import json
import logging
import os

logging.basicConfig(level=logging.INFO)
log = logging.getLogger(__name__)

NAMESPACE = os.getenv("NAMESPACE", "spark")
cmd = "kubectl --namespace {} get instances.kudo.dev -o json".format(NAMESPACE)

def delete_resource(api, resource):
    log.info("Deleting {} {}".format(api, resource))
    subprocess.call("kubectl --namespace {} delete {} {}".format(NAMESPACE, api, resource), shell=True)

subprocess.getoutput(cmd)
instances = json.loads(subprocess.getoutput(cmd))

versions = set()
for instance in instances["items"]:
    if instance["metadata"]["labels"]["kudo.dev/operator"] == "spark":
        versions.add(instance["spec"]["operatorVersion"]["name"])

        name = instance["metadata"]["name"]
        delete_resource("instance.kudo.dev", name)

for version in versions:
    delete_resource("operatorversion.kudo.dev", version)

delete_resource("operator.kudo.dev", "spark")
