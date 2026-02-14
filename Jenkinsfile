pipeline {
    agent any
    tools {
        go 'Go 1.24'
    }

    environment {
        // 核心调整：去掉私有仓库，只用本地镜像
        DATABASE_URL=credentials('DB_URL')
        SERVER_EXTERNAL_URL=credentials('SERVER_URL')
        GITHUB_CREDENTIAL_ID = "github-id"
        APP_NAME = "breez-lnurl"
        // 去掉私有仓库IP，只用本地镜像名
        DOCKER_IMAGE = "${APP_NAME}:latest"
        NAMESPACE = "lifpay-test"
        PORT = 4001
        BUILD_DIR = "./build"
    }

    stages {
        stage('拉取代码') {
            steps {
                git(url: 'https://github.com/ReaFstar/breez-lnurl.git', credentialsId: GITHUB_CREDENTIAL_ID, branch: 'main', changelog: true, poll: false)
                sh 'ls -al && cat go.mod'
            }
        }

        stage('项目编译') {
            steps {
                sh 'mkdir -p ${BUILD_DIR}'
                sh 'go mod tidy'
                sh 'CGO_ENABLED=0 GOOS=linux go build -o ${BUILD_DIR}/app ./'
                sh 'ls ${BUILD_DIR}/app'
            }
        }

        stage('构建本地Docker镜像') {
            steps {
                 sh 'ls ${BUILD_DIR}/app'
                 sh 'docker build -t ${DOCKER_IMAGE} .'
                 sh 'docker images | grep ${DOCKER_IMAGE}'
            }
        }

        stage('部署到k3s集群') {
            steps {
                 sh '''
                    # 部署k3s资源，重点改imagePullPolicy为Never（不拉取，只用本地镜像）
                    kubectl apply -f - <<EOF
                    apiVersion: apps/v1
                    kind: Deployment
                    metadata:
                      name: ${APP_NAME}
                      namespace: ${NAMESPACE}
                    spec:
                      replicas: 1
                      selector:
                        matchLabels:
                          app: ${APP_NAME}
                      template:
                        metadata:
                          labels:
                            app: ${APP_NAME}
                        spec:
                          containers:
                          - name: ${APP_NAME}
                            image: ${DOCKER_IMAGE}
                            # 关键：设置为Never，k3s不会去远程拉取，只用本地镜像
                            imagePullPolicy: Never
                            ports:
                            - containerPort: ${PORT}
                            env:
                            - name: DATABASE_URL
                              value: "${DATABASE_URL}"
                            - name: SERVER_EXTERNAL_URL
                              value: "${SERVER_EXTERNAL_URL}"
                            - name: SERVER_INTERNAL_URL
                              value: "http://${APP_NAME}-service:${PORT}"
                            livenessProbe:
                              tcpSocket:
                                port: ${PORT}
                              initialDelaySeconds: 10
                              periodSeconds: 5
                            readinessProbe:
                              tcpSocket:
                                port: ${PORT}
                              initialDelaySeconds: 5
                              periodSeconds: 3
                    ---
                    apiVersion: v1
                    kind: Service
                    metadata:
                      name: ${APP_NAME}-service
                      namespace: ${NAMESPACE}
                    spec:
                      type: NodePort
                      ports:
                      - port: ${PORT}
                        targetPort: ${PORT}
                        nodePort: 30401
                      selector:
                        app: ${APP_NAME}
                    EOF
                 '''
                 // 验证部署
                 sh 'kubectl get pods -n ${NAMESPACE} -l app=${APP_NAME}'
                 sh 'kubectl get svc -n ${NAMESPACE} ${APP_NAME}-service'
            }
        }
    }

    post {
        success {
            echo "===== 流水线执行成功！Go应用已部署到k3s集群 ====="
            sh '''
                echo "应用名称：${APP_NAME}"
                echo "访问地址：http://$(hostname -I | awk '{print $1}'):30401"
                echo "k3s命名空间：${NAMESPACE}"
            '''
        }
        failure {
            echo "===== 流水线执行失败！====="
            sh 'kubectl logs -n ${NAMESPACE} -l app=${APP_NAME} || true'
        }
        always {
            echo "清理临时文件"
            sh 'rm -rf ${BUILD_DIR}'
            // 可选：清理Docker镜像（测试环境）
            sh 'docker rmi ${DOCKER_IMAGE} || true'
        }
    }
}