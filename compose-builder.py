
# Tenemos que tener una idea de la identacion
# Podemos tener una variable que pasemos a las funciones
# con el numero de identacion a utilizar

class Writer:
    def __init__(self, config, output_file):
        self.config = config
        self.output_file = open(output_file, "w")

    def run(self):
        level = 0
        self.write_version(level)
        self.write_services(level)
        self.write_network(level)
        self.output_file.close()

    def write_version(self, level):
        self.write(level, "version: '3'")
    
    def write_network(self, level):
        self.write(level, "networks:")
        self.write(level+1, "tp2_net:")
        self.write(level+2, 'name: "tp2_net"')
        self.write(level+2, 'ipam:')
        self.write(level+3, 'driver: default')
        self.write(level+3, 'config:')
        self.write(level+4, '- subnet: 172.25.125.0/24')

    def write_services(self, level):
        self.write(level, "services:")
        level += 1
        self.write_rabbitmq(level)
        self.write(level, "")
        self.write_mom_admin(level)
        self.write(level, "")
        for (service, number) in self.config["worker_number"].items():
            if number == 1:
                self.write_service(level, service, self.config[service], id=None)
                self.write(level, "")
            else:
                for id in range(number):
                    self.write_service(level, service, self.config[service], id=id)
                    self.write(level, "")
            

    def write_rabbitmq(self, level):
        self.write(level, "rabbitmq:")
        self.write(level+1, "container_name: rabbitmq")
        self.write(level+1, "build:")
        self.write(level+2, "context: ./rabbitmq")
        self.write(level+2, "dockerfile: ./rabbitmq.dockerfile")
        self.write(level+1, "ports:")
        self.write(level+2, "- 15672:15672")
        self.write(level+1, "networks:")
        self.write(level+2, "- tp2_net")
        self.write(level+1, "volumes:")
        self.write(level+2, "- ./rabbitmq/config.conf:/etc/rabbitmq/rabbitmq.conf:ro")
        self.write(level+1, "healthcheck:")
        self.write(level+2, 'test: ["CMD", "curl", "-f", "http://localhost:15672"]')
        self.write(level+2, 'interval: 5s')
        self.write(level+2, 'timeout: 3s')
        self.write(level+2, 'retries: 5')


    def write_mom_admin(self, level):
        self.write(level, "mom-admin:")
        self.write(level+1, "build:")
        self.write(level+2, "context: ./")
        self.write(level+2, "dockerfile: ./server/common/message_middleware/Dockerfile")
        self.write(level+1, "container_name: mom-admin")
        self.write(level+1, "entrypoint: /admin")
        self.write(level+1, "restart: on-failure")
        self.write(level+1, "depends_on:")
        self.write(level+2, "rabbitmq:")
        self.write(level+3, "condition: service_healthy")
        self.write(level+1, "links:")
        self.write(level+2, "- rabbitmq")
        self.write(level+1, "networks:")
        self.write(level+2, "- tp2_net")

    def write_service(self, level, name, config, id=None):
        entrypoint = config["entrypoint"]
        dockerfile = config["dockerfile"]
        environment = config["environment"] 
        if id is not None:
            name = f"{name}{id}"
            environment.append(f"ID={id}") 
        else:
            environment.append(f"ID=0") 

        self.write(level, f"{name}:")
        self.write(level+1, "build:")
        self.write(level+2, "context: ./")
        self.write(level+2, f"dockerfile: {dockerfile}")
        self.write(level+1, f"container_name: {name}")
        self.write(level+1, f"entrypoint: {entrypoint}")
        self.write(level+1, "restart: on-failure")
        self.write(level+1, "depends_on:")
        self.write(level+2, "rabbitmq:")
        self.write(level+3, "condition: service_healthy")
        self.write(level+1, "links:")
        self.write(level+2, "- rabbitmq")
        self.write(level+1, "networks:")
        self.write(level+2, "- tp2_net")
        self.write(level+1, "volumes:")
        self.write(level+2, "- ./config.json:/config.json")
        self.write_environments(level+1, environment)


    def write_environments(self, level, environment):
        self.write(level, "environment:")
        level += 1
        for env in environment:
            self.write(level, f"- {env}")

    def write(self, level, string):
        space = "  " * level
        to_write = f"{space}{string}\n"
        self.output_file.write(to_write)

def main():

    worker_number = {
        "admin": 1,
        "post-score-adder": 1,
        "post-digestor": 2,
        "post-score-avg-calculator": 1,
        "post-above-avg-filter": 2,
        "best-sentiment-avg-downloader": 1,
        "sentiment-joiner": 2,
        "student-joiner": 2,
        "comment-digestor": 2,
        "post-sentiment-avg-calculator": 2,
        "student-comment-filter": 2,
    }

    config = {
        "admin": {
            "entrypoint": "/admin",
            "dockerfile": "./server/admin/Dockerfile",
            "environment": [
                "PROCESS_GROUP=admin"
            ]
        },
        "post-digestor": {
            "entrypoint": "/post_digestor",
            "dockerfile": "./server/post_digestor/Dockerfile",
            "environment": [
                f"LOAD_BALANCE={worker_number['sentiment-joiner']}",
                "PROCESS_GROUP=post_digestor"
            ]
        },
        "post-score-adder": {
            "entrypoint": "/post_score_adder",
            "dockerfile": "./server/post_score_adder/Dockerfile",
            "environment": [
                "PROCESS_GROUP=post_score_adder"
            ]
        },
        "post-score-avg-calculator": {
            "entrypoint": "/calculator",
            "dockerfile": "./server/post_score_avg_calculator/Dockerfile",
            "environment": [
                "PROCESS_GROUP=post_score_avg_calculator"
            ]
        },
        "post-above-avg-filter": {
            "entrypoint": "/filter",
            "dockerfile": "./server/post_above_avg_filter/Dockerfile",
            "environment": [
                f"LOAD_BALANCE={worker_number['student-joiner']}",
                "PROCESS_GROUP=post_above_avg_filter"
            ]
        },
        "comment-digestor": {
            "entrypoint": "/comment_digestor",
            "dockerfile": "./server/comment_digestor/Dockerfile",
            "environment": [
                f"LOAD_BALANCE={worker_number['post-sentiment-avg-calculator']}",
                "PROCESS_GROUP=comment_digestor"
            ]
        },
        "post-sentiment-avg-calculator": {
            "entrypoint": "/calculator",
            "dockerfile": "./server/post_sentiment_avg_calculator/Dockerfile",
            "environment": [
                f"LOAD_BALANCE={worker_number['sentiment-joiner']}", 
                "PROCESS_GROUP=post_sentiment_avg_calculator"
            ]
        },
        "sentiment-joiner": {
            "entrypoint": "/joiner",
            "dockerfile": "./server/joiner/Dockerfile",
            "environment": [
                "PROCESS_GROUP=sentiment_joiner"
            ]
        },
        "student-joiner": {
            "entrypoint": "/joiner",
            "dockerfile": "./server/joiner/Dockerfile",
            "environment": [
                "PROCESS_GROUP=student_joiner"
            ]
        },
        "student-comment-filter": {
            "entrypoint": "/filter",
            "dockerfile": "./server/student_comment_filter/Dockerfile",
            "environment": [
                f"LOAD_BALANCE={worker_number['student-joiner']}", 
                "PROCESS_GROUP=student_comment_filter"
            ]
        },
        "best-sentiment-avg-downloader": {
            "entrypoint": "/downloader",
            "dockerfile": "./server/best_sentiment_avg_downloader/Dockerfile",
            "environment": [
                "PROCESS_GROUP=best_sentiment_avg_downloader"
            ]
        },
        "worker_number": worker_number
    }

    output_path = "./docker-compose-server.yaml"

    writer = Writer(config, output_path)
    writer.run()

main()