import pika
import json
import tensorflow as tf
import numpy as np

import os
os.environ['CUDA_VISIBLE_DEVICES'] = '-1'
from PIL import Image
print("The script is running!")
model = tf.keras.applications.MobileNetV2(weights="imagenet")

def process_image(image_path):
    image = Image.open(image_path).resize((224, 224))
    image_array = np.expand_dims(np.array(image) / 255.0, axis=0)
    predictions = model.predict(image_array)
    decoded_predictions = tf.keras.applications.mobilenet_v2.decode_predictions(predictions, top=5)
    keywords = [item[1] for item in decoded_predictions[0]]
    return keywords

def callback(ch, method, properties, body):
    message = json.loads(body)
    image_path = message['image_path']

    keywords = process_image(image_path)

    response = {"image_id": message['image_id'], "keywords": keywords}
    ch.basic_publish(exchange='', routing_key='response_queue', body=json.dumps(response))
    print(f"Processed image {message['image_id']} and sent response.")


connection = pika.BlockingConnection(
    pika.ConnectionParameters(
        host=os.getenv("RABBITMQ_HOST"),
        port=5672,
        credentials=pika.PlainCredentials(
            username=os.getenv("RABBITMQ_USER", "guest"),
            password=os.getenv("RABBITMQ_PASS", "guest")
        )
    )
)
channel = connection.channel()

# Declare queues
channel.queue_declare(queue='request_queue')
channel.queue_declare(queue='response_queue')

# Consume messages from the request queue
channel.basic_consume(queue='request_queue', on_message_callback=callback, auto_ack=True)

print("Python Service is waiting for messages...")
channel.start_consuming()
