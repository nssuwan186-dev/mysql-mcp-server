# Minimal SSH bastion for integration tests (key-based auth only).
FROM alpine:3.19

RUN apk add --no-cache openssh-server openssh-client \
    && mkdir -p /root/.ssh \
    && chmod 0700 /root/.ssh

# Deterministic ed25519 host key (matches tests/integration/fixtures/ssh_bastion_known_hosts).
COPY ssh_host_ed25519_key /etc/ssh/ssh_host_ed25519_key
COPY ssh_host_ed25519_key.pub /etc/ssh/ssh_host_ed25519_key.pub
RUN chmod 600 /etc/ssh/ssh_host_ed25519_key \
    && chmod 644 /etc/ssh/ssh_host_ed25519_key.pub

# RSA/ECDSA for sshd defaults (ed25519 is fixed above; -A only creates missing types).
RUN ssh-keygen -A

# Copy test public key so we can connect with the matching private key.
COPY authorized_keys /root/.ssh/authorized_keys
RUN chmod 0600 /root/.ssh/authorized_keys

# Allow root login with public key only.
RUN echo "PermitRootLogin prohibit-password" >> /etc/ssh/sshd_config \
    && echo "PasswordAuthentication no" >> /etc/ssh/sshd_config \
    && echo "KbdInteractiveAuthentication no" >> /etc/ssh/sshd_config

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
