# Minimal SSH bastion for integration tests (key-based auth only).
FROM alpine:3.19

RUN apk add --no-cache openssh-server openssh-client \
    && ssh-keygen -A \
    && mkdir -p /root/.ssh \
    && chmod 0700 /root/.ssh

# Copy test public key so we can connect with the matching private key.
COPY authorized_keys /root/.ssh/authorized_keys
RUN chmod 0600 /root/.ssh/authorized_keys

# Allow root login with public key only.
RUN echo "PermitRootLogin prohibit-password" >> /etc/ssh/sshd_config \
    && echo "PasswordAuthentication no" >> /etc/ssh/sshd_config \
    && echo "KbdInteractiveAuthentication no" >> /etc/ssh/sshd_config

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
