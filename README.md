# Trabajo Práctico 2 

# "Middleware y Coordinación de Procesos"

## Sistemas Distribuidos I



#### Servidor

El comando para correr el servidor es el siguiente:


```bash
make server-up
```



Para finalizar su ejecución, el comando es:


```bash
make server-down
```



El servidor cuenta con un archivo de configuraciones `config.json` que puede ser modificado para alterar ciertos parámetros de su funcionalidad.

Adicionalmente, se incluye un script de python `compose-builder.py` que permite especificar el número de procesos a ejecutar para cada etapa del pipeline. Este script generará un `docker-compose-server.yaml` acorde a las configuraciones especificadas, que será utilizado en el comando `make server-up`  para la inicialización del servidor.



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
  post: "1us"	
  comment: "1us"

files-path:	//Path a los archivos de posts y comentarios
  post: "./files/posts_small.csv"
  comment: "./files/comments_small.csv"

log:
  level: "info"
```



**Nota**: Tanto el servidor como el cliente quedarán a la escucha del correspondiente comando de finalización para terminar su ejecución.

