
FROM python:3.9-slim


WORKDIR /app

RUN pip install pika tensorflow pillow


COPY . .

# Expose the port the service runs on
EXPOSE 5000

# Command to run the application
CMD ["python", "init.py"]
