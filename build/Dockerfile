FROM centos:7
MAINTAINER Jason Montleon

ENV USER_NAME=openstackbroker \
    USER_UID=1001 \
    BASE_DIR=/opt/openstackbroker
ENV HOME=${BASE_DIR}

RUN mkdir -p ${BASE_DIR} ${BASE_DIR}/etc \
 && useradd -u ${USER_UID} -r -g 0 -M -d ${BASE_DIR} -b ${BASE_DIR} -s /sbin/nologin -c "openstackbroker user" ${USER_NAME} \
 && chown -R ${USER_NAME}:0 ${BASE_DIR} \
 && chmod -R g+rw ${BASE_DIR} /etc/passwd


RUN yum -y update \
 && yum -y install epel-release centos-release-openshift-origin \
 && yum -y install origin-clients net-tools bind-utils \
 && yum clean all

RUN mkdir /var/log/ansible-service-broker \
    && touch /var/log/ansible-service-broker/asb.log \
    && mkdir /etc/ansible-service-broker

COPY entrypoint.sh /usr/bin/
COPY openstackbroker /usr/bin/obd

RUN chown -R ${USER_NAME}:0 /var/log/ansible-service-broker \
 && chown -R ${USER_NAME}:0 /etc/ansible-service-broker \
 && chmod -R g+rw /var/log/ansible-service-broker /etc/ansible-service-broker

USER ${USER_UID}
RUN sed "s@${USER_NAME}:x:${USER_UID}:@${USER_NAME}:x:\${USER_ID}:@g" /etc/passwd > ${BASE_DIR}/etc/passwd.template

ENTRYPOINT ["entrypoint.sh"]
