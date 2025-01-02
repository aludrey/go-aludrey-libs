# go-aludrey-libs
Golang aludrey Libs

Pre-Commit

1) brew install pre-commit
2) pre-commit install


## Queue:

#### func SendMessage(ctx context.Context, queueURL string, message string, delaySeconds int64) error
----
  
### Parámetros:
- ctx: Contexto de la operación.
- queueURL: URL de la cola.
- message: Cuerpo del mensaje a enviar.
- delaySeconds: Tiempo de retraso en segundos antes de que el mensaje esté disponible. Puede ser 0 para ningún retraso.
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
SendMessage envía un mensaje a una cola con un posible retraso antes de estar disponible para su consumo.
Retorna un error si la operación de envío falla.

### Ejemplo
        file, err := aludrey_queue.SendMessage(context.Background(), "http://aludrey-dev-sqs-test", "Hello, I'm an example!", 5)

Envia el mensaje <i>Hello, I'm an example!</i> a la cola con URL <i>http://aludrey-dev-sqs-test</i> con un delay de <i>5</i> segundos.

----
#### func DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error
----
  
### Parámetros:
- ctx: Contexto de la operación.
- queueURL: URL de la cola.
- receiptHandle: Handle del recibo del mensaje que se va a eliminar.
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
DeleteMessage elimina un mensaje específico de una cola.
Retorna un error si la operación de eliminación falla.

### Ejemplo
        err := aludrey_queue.DeleteMessage(context.Background(), "http://aludrey-dev-sqs-test", "receipt_handle_test")

Elimina el mensaje con receiptHandle <i>receipt_handle_test</i> de la cola con URL <i>http://aludrey-dev-sqs-test</i>

----
#### func GetMessages(ctx context.Context, queueURL string, maxNumberOfMessages int64, visibilityTimeout int64) ([]*sqs.Message, error)
----
  
### Parámetros:
- ctx: Contexto de la operación.
- queueURL: URL de la cola.
- maxNumberOfMessages: Número máximo de mensajes para recuperar.
- visibilityTimeout: Tiempo en segundos durante el cual el mensaje no estará disponible para ser recuperado nuevamente.
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
GetMessages recupera mensajes de una cola.
Retorna un slice de mensajes y un error si la operación de recuperación falla.

### Ejemplo
        messages, err := aludrey_queue.GetMessages(context.Background(), "http://aludrey-dev-sqs-test", 5, 30)

Recibe una cantidad máxima de <i>10</i> mensajes de la cola con URL <i>http://aludrey-dev-sqs-test</i> con un tiempo de visibilidad de <i>30</i> segundos.




----
#### func GetMessagesAvailable(queueURL) (int64, error)
----
  
### Parámetros:
- queueURL: URL de la cola de SQS a leer.
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
GetMessagesAvailable cuenta mensajes disponibles en una cola de SQS.
Retorna cantidad de mensajes y un error si la operación de recuperación falla.

### Ejemplo
        cantMsg, err := aludrey_queue.GetMessagesAvailable("http://aludrey-dev-sqs-test")


## Bucket:

#### DownloadFile(ctx context.Context, bucketName string, itemFile string) (*os.File, error)</b>
----
  
### Parámetros:
- ctx: Contexto de la operación
- bucketName: Nombre del bucket del que se va a ejecutar la operación
- itemFile: Archivo que se va a descarcar, incluye el camino dentro del bucket
  
### Devuelve:
- Referencia al archivo descargado
- Error en caso de no poder completar la operación
  
### Descripción:
Intenta descargar el archivo descrito en <i>itemFile</i> del bucket con nombre <i>bucketName</i> y devuelve una referencia al archivo descargado o error en caso de no poder completar la operación

### Ejemplo
        file, err := aludrey_bucket.DownloadFile(context.Background(), "ejemplo-s3", "subdir/test.txt")

Descarga el archivo <i>text.txt</i> con el mismo nombre desde el bucket <i>ejemplo-s3</i> y devuelve la referencia al archivo

----
#### DeleteFile(ctx context.Context, bucketName string, itemFile string) error
----
  
### Parámetros:
- ctx: Contexto de la operación
- bucketName: Nombre del bucket del que se va a ejecutar la operación
- itemFile: Archivo que se intenta eliminar, incluye el camino dentro del bucket
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
Intenta eliminar el archivo descrito en <i>itemFile</i> del bucket con nombre <i>bucketName</i> y devuelve un error en caso de no poder completar la operación

### Ejemplo
        err := aludrey_bucket.DeleteFile(context.Background(), "ejemplo-s3", "subdir/test.txt")

Elimina el archivo <i>text.txt</i> en la carpeta <i>subdir</i> en el bucket <i>ejemplo-s3</i>

----
#### UploadFile(ctx context.Context, bucketName string, localItem string, itemFile string) error
----
  
### Parámetros:
- ctx: Contexto de la operación
- bucketName: Nombre del bucket del que se va a ejecutar la operación
- localItem: Ruta, en el sistema de archivos local, del archivo que se desea subir, puede ser relativa o absoluta
- itemFile: Ruta destino en el bucket hacia el que se va a ejecutar la operación
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
Intenta subir el archivo descrito por <i>localItem</i> al bucket con nombre <i>bucketName</i> y ruta del archivo en el bucket. Devuelve error en caso de no completar la operación

### Ejemplo
        err := aludrey_bucket.UploadFile(context.Background(), "ejemplo-s3", "subdir/test.txt", "bucketsubdir/test.txt")

Lee el archivo <i>subdir/test.txt</i> y lo sube para el bucket <i>ejemplo-s3</i> con ruta <i>bucketsubdir/test.txt</i>

----
#### ListFiles(ctx context.Context, bucketName string, itemFile string) ([]string, error)
----
  
### Parámetros:
- ctx: Contexto de la operación
- bucketName: Nombre del bucket del que se va a ejecutar la operación
- itemFile: Ruta dentro del bucket de la cual se van a listar los archivos
  
### Devuelve:
- Lista de archivos
- Error en caso de no poder completar la operación
  
### Descripción:
Lee del bucket <i>bucketName</i> los archivos en la ruta <i>itemFile</i> y devuelve la lista de archivos en caso de existir. Devuelve error en caso de no poder leer de la ruta especificada

### Ejemplo
        err := aludrey_bucket.ListFiles(context.Background(), "ejemplo-s3", "subdir/")

Obtiene la lista de archivos en la ruta <i>subdir/</i> del bucket <i>ejemplo-s3</i>

----
#### CopyFile(ctx context.Context, bucketName string, itemFile string, destBucketName string, destItemFile string) error
----
  
### Parámetros:
- ctx: Contexto de la operación
- bucketName: Nombre del bucket de origen
- itemFile: Ruta, dentro del bucket de origen, del archivo que se va a copiar
- destBucketName: Nombre del bucket de destino
- destItemFile: Ruta, dentro del bucket de destino, del archivo copiado
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
Copia del bucket <i>bucketName</i> el contenido del archivo <i>itemFile</i> al bucket <i>destBucketName</i> con ruta <i>destItemFile</i>

### Ejemplo
        err := aludrey_bucket.CopyFile(context.Background(), "ejemplo-s3", "test.txt", "ejemplo-s3-destino", "text.txt")

Copia el archivo <i>test.txt</i> del bucket <i>ejemplo-s3</i> al archivo <i>text.txt</i> en el bucket <i>ejemplo-s3-destino</i>

----
#### MoveFile(ctx context.Context, bucketName string, itemFile string, destBucketName string, destItemFile string) error
----
  
### Parámetros:
- ctx: Contexto de la operación
- bucketName: Nombre del bucket de origen
- itemFile: Ruta, dentro del bucket de origen, del archivo que se va a mover
- destBucketName: Nombre del bucket de destino
- destItemFile: Ruta, dentro del bucket de destino, del archivo movido
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
Mueve del bucket <i>bucketName</i> el contenido del archivo <i>itemFile</i> al bucket <i>destBucketName</i> con ruta <i>destItemFile</i>

### Ejemplo
        err := aludrey_bucket.MoveFile(context.Background(), "ejemplo-s3", "test.txt", "ejemplo-s3-destino", "text.txt")

Mueve el archivo <i>test.txt</i> del bucket <i>ejemplo-s3</i> al archivo <i>text.txt</i> en el bucket <i>ejemplo-s3-destino</i>

## Parameter:

#### func LoadParameters(evironment string, appName string) error
----
  
### Parámetros:
- evironment: Se refiere al ambiente de despliegue.
- appName: Nombre de la aplicación.
  
### Devuelve:
- Error en caso de no poder completar la operación
  
### Descripción:
LoadParameters busca en los Parameter Store de la nube aquellos cuyo nombre contengan el prefijo formado por los parámetros <i>environment</i> y <i>appName</i>. Así, si un parámetro se encuentra almacenado como <i>/DEV/TEST_APP/PORT</i>, la función deberá ser llamada <i>parameter.LoadParameters("DEV", "TEST_APP")</i> y creará la variable de entorno <i>PORT</i> con el valor asociado en el <i>Parameter Store</i>

### Ejemplo
        err := parameter.LoadParameters("DEV", "TEST_APP")

Crea las variables de entorno bajo el camino <i>/DEV/TEST_APP</i> con el nombre y valor asignados en el Parameter Store.

----
#### CreateParameter(evironment string, appName string, name string, value string) error
----

### Parámetros
- evironment: Se refiere al ambiente de despliegue.
- appName: Nombre de la aplicación.
- name: Es la llave del parámetro
- value: El valor del parámetro

### Devuelve
- Error en caso de no poder completar la operación

### Descripción
CreateParameter crea un parámetro en el Parameter Store con el prefijo formado por <i>environment</i>, <i>appName</i> y <i>name</i> con valor <i>value</i>

### Ejemplo
        err := parameter.CreateParameter("DEV", "TEST_APP", "PORT", "8080")

Crea un parámetro con prefijo <i>/DEV/TEST_APP/PORT</i> y valor <i>8080</i>

----
#### DeleteParameter(evironment string, appName string, name string) error
----

### Parámetros
- evironment: Se refiere al ambiente de despliegue.
- appName: Nombre de la aplicación.
- name: Es la llave del parámetro.

### Devuelve
- Error en caso de no poder completar la operación

### Descripción
DeleteParameter elimina el parámetro en el Parameter Store con el prefijo formado por <i>environment</i>, <i>appName</i> y <i>name</i>

### Ejemplo
        err := parameter.DeleteParameter("DEV", "TEST_APP", "PORT")

Elimina el parámetro con prefijo <i>/DEV/TEST_APP/PORT</i>

----
#### UpdateParameter(evironment string, appName string, name string, value string) error
----

### Parámetros
- evironment: Se refiere al ambiente de despliegue.
- appName: Nombre de la aplicación.
- name: Es la llave del parámetro
- value: El valor del parámetro

### Devuelve
- Error en caso de no poder completar la operación

### Descripción
UpdateParameter actualiza el valor del parámetro en el Parameter Store con el prefijo formado por <i>environment</i>, <i>appName</i> y <i>name</i> con valor <i>value</i>

### Ejemplo
        err := parameter.UpdateParameter("DEV", "TEST_APP", "PORT", "8080")

Modifica el parámetro con prefijo <i>/DEV/TEST_APP/PORT</i> con valor <i>8080</i>

## Database:

#### func FindById(ctx context.Context, tableName string, id string, itemType interface{}) (interface{}, error)
----
  
### Parámetros:
- ctx: El contexto de la aplicación.
- tableName: Nombre de la tabla.
- id: Id de la búsqueda
- itemType: El tipo de elemento al que se va a mapear la consulta
  
### Devuelve:
- Devuelve un objeto del tipo del parámetro <i>itemType</i> con los valores de la consulta o un error en caso de no poder completar la operación
  
### Descripción:
FindById realiza una consulta a la entidad <i>tableName</i> buscando el registro con id <i>id</i> y retorna los valores, en caso de encontrarlos, en un objeto del tipo pasado en <i>itemType</i>. En caso de no completar la operación o no encontrar el <i>id</i> devuelve un error

### Ejemplo
        mockItem := &Item{
		Id:   "",
		Name: "",
	}
        item, err := FindById(ctx, "test_table", "1", mockItem)

Busca en <i>test_table</i> el registro con <i>id=1</i>

----
#### func FindAll(ctx context.Context, tableName string, itemType interface{}) ([]interface{}, error)
----
  
### Parámetros:
- ctx: El contexto de la aplicación.
- tableName: Nombre de la tabla.
- itemType: El tipo de elemento al que se va a mapear la consulta.
  
### Devuelve:
- Devuelve una lista de objetos del tipo del parámetro <i>itemType</i> con los valores de la consulta o un error en caso de no poder completar la operación
  
### Descripción:
FindAll realiza una consulta a la entidad <i>tableName</i> y retorna todos los valores encontrados en una lista de objetos del tipo pasado en <i>itemType</i>. En caso de no completar la operación devuelve un error

### Ejemplo
        mockItem := &Item{
		Id:   "",
		Name: "",
	}
        item, err := FindAll(ctx, "test_table", mockItem)

Obtiene todos los registros de la table <i>test_table</i>

----
#### func PutItem(ctx context.Context, tableName string, item interface{}) error
----
  
### Parámetros:
- ctx: El contexto de la aplicación.
- tableName: Nombre de la tabla.
- item: El registro a insertar.
  
### Devuelve:
- Devuelve error en caso de no completar la operación.
  
### Descripción:
PutItem agrega el registro descrito en <i>item</i> a la tabla <i>tableName</i>

### Ejemplo
        mockItem := &Item{
		Id:   "5",
		Name: "Prueba",
	}
        err := PutItem(ctx, "test_table", mockItem)

Agrega el registro descrito en <i>mockItem</i> a la tabla <i>test_table</i>

----
#### func DeleteItem(ctx context.Context, tableName string, id string) error
----
  
### Parámetros:
- ctx: El contexto de la aplicación.
- tableName: Nombre de la tabla.
- id: El id del registro de la operación.
  
### Devuelve:
- Devuelve error en caso de no completar la operación.
  
### Descripción:
DeleteItem elimina el registro con id <i>id</i> de la tabla <i>tableName</i>

### Ejemplo
        err := DeleteItem(ctx, "test_table", "5")

Elimina el registro con <i>id=5</i> de la tabla <i>test_table</i>


## Notifier
----
#### func WithPrefix(prefix string)
----
  
### Parámetros:
- prefix: prefijo para los mensajes

### Descripción:
Configura <i>prefix</i> como prefijo para todos los mensajes

### Ejemplo
       n Notifier
       n.WithPrefix("Error de facturación")
----
#### func Notify(msg string) error
----
### Parámetros:
- msg: mensaje a notificar
  
### Devuelve:
- Devuelve error en caso de no completar la operación.
  
### Descripción:
Notifica el mensaje <i>msg</i> con el prefijo configurado

### Ejemplo
        err := Notify("Error al facturar")

## GoogleChatNotifier
----
#### func NewGoogleChatNotifier(webhookURL string) Notifier
----
### Parámetros:
- msg: webhookURL: URL del webhook de Google Chat
  
### Devuelve:
- Devuelve un objeto Notifier
  
### Descripción:
Crea un nuevo objeto Notifier para enviar mensajes a Google Chat

### Ejemplo
        n := NewGoogleChatNotifier("https://chat.googleapis.com/...")


## Logger
----
#### func SetLevel(level Level)
----
### Parámetros:
- level: Nivel de log a configurar
### Descripción:
Configura el nivel de log a mostrar en el stream de logs
### Ejemplo
        logger.SetLevel(InfoLevel)
----
#### func WithFields(keyValues Fields) Logger
----
### Parámetros:
- keyValues: Mapa de campos a agregar al log
### Devuelve:
- Un nuevo Logger con los campos agregados
### Descripción:
Agrega campos al log, estos se agregarán solo en la próxima entrada
### Ejemplo
        n := logger.WithFields(Fields{"key1": "value1", "key2": "value2"})
----
#### func WithRequestId(requestId string) Logger
----
### Parámetros:
- requestId: ID de la solicitud
### Devuelve:
- Un nuevo Logger con el ID de solicitud agregado
### Descripción:
Agrega el ID de solicitud al log, este se agregará en todas las futuras entradas
#### Ejemplo
        n := logger.WithRequestId("fafafa")
----
#### func WithStreamName(streamName string) Logger
----
### Parámetros:
- streamName: Nombre del stream
### Devuelve:
- Un nuevo Logger con el nombre del stream agregado
### Descripción:
Agrega el nombre del stream al log, este se agregará en todas las futuras entradas
### Ejemplo
        n := logger.WithStreamName("stream1")

----
#### func Panic(args ...interface{})
----
#### func Fatal(args ...interface{})
----
#### func Error(args ...interface{})
----
#### func Warn(args ...interface{})
----
#### func Info(args ...interface{})
----
#### func Debug(args ...interface{})
----
#### func Trace(args ...interface{})
----
#### func Print(args ...interface{})
----
### Parámetros:
- Cualquier cantidad de argumentos, intentará imprimirlos sin importar el tipo

### Descripción:
Agrega un mensaje al log con el nivel correspondiente, en el stream de logs configurado

### Ejemplo
        logger.Info("Mensaje de información")

## LogrusLogger
#### func NewLogger(config LogrusLoggerConfig) Logger
----
### Parámetros:
- config: Configuración del logger
### Devuelve:
- Un nuevo Logger
### Descripción:
Crea un nuevo Logger con la configuración dada. Es compatible con firehose, aunque se le pueden agregar más hooks en el futuro
### Ejemplo
    var logger Logger = NewLogger(LogrusLoggerConfig{
        Level:        InfoLevel,
        ReportCaller: true,
        FirehoseHook: &loggerHooks.FirehoseHookConfig{
            DefaultStreamName: "aludrey-dev-us-e2-logs-apps",
            Env:               "dev",
            AppName:           "library-test",
            AwsRegion:         "us-east-2",
        },
    })