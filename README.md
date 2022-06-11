# Trabajo Práctico 2 

# "Middleware y Coordinación de Procesos"

## Sistemas Distribuidos I



#### Acerca del trabajo

Se trata de un trabajo práctico desarrollado para la materia Sistemas Distribuidos I de la Facultad de Ingeniería de la Universidad de Buenos Aires que consiste en el procesamiento de un stream de datos siguiendo una arquitectura distribuida multicomputing en forma de pipeline.

Los datos procesados corresponden a un conjunto de posts y comentarios extraidos de reddit.

Como su nombre lo indica, este trabajo hace hincapié en la coordinación de los procesos que integran el servidor y el desarrollo de un middleware que abstraiga lógica de conunicación entre estos.

El mismo fue implementado en Golang, utilizando RabbitMQ como middleware de mensajes.


#### Primeros pasos

Para la ejecución del trabajo será necesaria la instalación de golang versión 1.17+ y python 3.8.10+

Luego, sobre la carpeta raíz del respositorio clonado, deberán instalarse las dependencias del proyecto:


```bash
go mod vendor
```


Las dependencias utilizadas son:

* jsonparser: Como biblioteca de parseo de configuraciones
* viper: Como biblioteca de parseo de configuraciones y variables de entorno
* logrus: Como biblioteca de logging
* amqp: Como biblioteca para la comunicación con el servidor de RabbitMQ


La versión utilizada de cada una de ellas puede obtenerse del archivo `go.mod`



#### Servidor

El comando para correr el servidor es el siguiente:


```bash
make server-up
```



Para finalizar su ejecución, el comando es:


```bash
make server-down
```



El servidor cuenta con dos archivos de configuraciones `config.json`  y `launch.json` que pueden ser modificados para alterar ciertos parámetros de su funcionalidad.

El archivo `config.json` tiene como propósito permitir modificar ciertos parámetros internos de cada proceso que se ejecutará en el servidor. El mismo cuenta con el siguiente formato:


```
{
    "general": { //Configuración general a todos los procesos
        "mom_address": "amqp://rabbitmq" //Dirección del servicio de RabbitMQ
    },
    "admin": { //Configuración particular del proceso
        "process_name": "Admin", //Nombre que identifica al proceso
        "log_level": "info", //Nivel de logueo deseado para el proceso
        "server_address": "admin:12345", //Dirección sobre la cual escuchará el servidor
        "mom_msg_batch_timeout": "5ms", //Tiempo de espera máximo para mensajes batch en el MOM
        "mom_msg_batch_target_size": 1000000, //Tamaño deseado de batch de mensajes en el MOM
        "mom_channel_buffer_size": 10000, //Tamaño de los buffers internos del MOM
        "queues": { //Definición de las colas que poseerá el proceso
            "average_result": {
                "name": "", //Nombre de la cola
                "class": "fanout", //Tipo de cola
                "topic": "", //Topico desde el cuál escuchará
                "source": "post_score_avg_result", //A qué cola se unirá para obtener mensajes
                "direction": "read" //Sentido del flujo de datos (escritura/lectura)
            },
            ...
        }
    },
  ...
}
```



El archivo `launch.json` tiene como propósito modificar parámetros previos a la ejecución de los procesos del servidor, como pueden ser el número de procesos involucrados o variables de entorno. El mismo cuenta con el siguiente formato:

```
{
    "worker_number": { //Número de procesos a lanzar de cada tipo
        "post-score-adder": 1,
        "post-digestor": 2,
        "post-above-avg-filter": 1,
        "sentiment-joiner": 2,
        "student-joiner": 1,
        "comment-digestor": 2,
        "post-sentiment-avg-calculator": 1,
        "student-comment-filter": 2
    },
    "workers": {
        "admin": {
            "entrypoint": "/admin", //Punto de entrada en el contenedor
            "dockerfile": "./server/admin/Dockerfile", //Path al Dockerfile del servicio
            "environment": [ //Variables de entorno
                "PROCESS_GROUP=admin" //Nombre del proceso, útil para localizar configuraciones.
            ]
        },
        "post-digestor": {
            "entrypoint": "/post_digestor",
            "dockerfile": "./server/post_digestor/Dockerfile",
            "environment": [
                "LOAD_BALANCE=sentiment-joiner", //Indica el proceso sobre el cuál se hará load balance
                "PROCESS_GROUP=post_digestor"
            ]
        },
     ...
 	}
 }
```

Aquellos procesos que no pueden ser replicados han sido excluidos de la configuración de worker_number

Para el procesamiento de este archivo se incluye un srcipt de python  `compose-builder.py` el cuál generará un `docker-compose-server.yaml` acorde a las configuraciones especificadas, que será utilizado en el comando `make server-up`  para la inicialización del servidor.



### Cliente

Se incluye una aplicación cliente, que tiene como propósito el envío del stream de datos al servidor. 

El cliente deberá ser inicializado únicamente cuando el servidor se encuentre a la espera de una conexión.

El comando para su ejecución es el siguiente:


```bash
make client-up
```

Para su finalización el comando es:


```bash
make client-down
```


Además, el cliente cuenta con un archivo de configuraciones `/client/config.yaml` desde el cual es posible alterar parte de su funcionamiento.



*Archivo de Configuraciones*

```yaml
server:
  address: "admin:12345"

loop-period: //Frecuencia de envío de posts y comentarios
  post: "100ns"	
  comment: "100ns"

files-path:	//Path a los archivos
  post: "./files/posts_medium.csv" //De donde se leeran los posts
  comment: "./files/comments_medium.csv" //De donde se leeran los comentarios
  sentiment-meme: "./files/sentiment_meme" //En donde se almacenará el meme descargado
  school-memes: "./files/school_memes" //En dónde se almacenaran las urls de memes escolares

log:
  level: "info"
```



### Archivos de entrada

Tanto el cliente como el servidor admiten posts y comentarios leidos en formato csv.

El dataset en cuestión fue extraído de [kaggle](https://www.kaggle.com/datasets/pavellexyr/the-reddit-irl-dataset ) y modificado para soportar un separador de colúmnas distinto `0x1f`.

Se provee una [carpeta de Drive](https://drive.google.com/drive/folders/1xi8uC3BLSM6CgGtrrNcpwrqDVihYh2k_?usp=sharing) en la cuál se proponen distintos subsets de datos formados a partír del original:

* Full: Todos los posts y comentarios (aprox 3.300.000 posts y 12.000.000 comentarios)
* Big: 2.700.000 posts y 10.000.000 comentarios
* Upper Half: 1.600.000 posts y 6.000.000 comentarios, a partir de la mitad del dataset.
* Half: 1.600.000 posts y 6.000.000 comentarios.
* Medium: 1.100.000 posts y 4.000.000 comentarios.
* Small: 550.000 posts y 2.000.000 comentarios.



Adicionalmente se provee un notebook `reddit-mees-analysis.ipynb` que podrá utilizarse para obtener nuevos subsets de datos o realizar validaciones sobre los resultados obtenidos por el servidor.

Al final del mismo se detallan los resultados obtenidos para cada métrica calculada para los subsets propuestos.



### Notas

- Tanto el servidor como el cliente quedarán a la escucha del correspondiente comando de finalización para terminar su ejecución.
