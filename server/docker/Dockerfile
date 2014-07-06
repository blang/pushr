FROM blang/supervisor
ENV DEBIAN_FRONTEND noninteractive

RUN mkdir /data
ADD ./server /usr/local/sbin/pushr
ADD ./run.sh /usr/local/sbin/pushr.run

# Define mountable directories.
VOLUME ["/data"]

# Define working directory.
WORKDIR /data

# Expose ports.
EXPOSE 7000

ADD pushr.sv.conf /etc/supervisor/conf.d/

CMD ["supervisord"]