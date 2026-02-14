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
        DOCKER_IMAGE = "${APP_NAME}:v1.0"
        NAMESPACE = "lifpay-test"
        PORT = 4001
        BUILD_DIR = "./build"
    }

    stages {
        stage('拉取代码') {
            steps {
                checkout scm
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
                    # 1. 替换deploy.yaml里的占位符（用Jenkins的环境变量）
                    sed -i "s|__DATABASE_URL__|${DATABASE_URL}|g" deploy.yaml
                    sed -i "s|__SERVER_EXTERNAL_URL__|${SERVER_EXTERNAL_URL}|g" deploy.yaml

                    # 2. 执行部署（用文件方式，避免内嵌YAML解析问题）
                    kubectl apply -f deploy.yaml -n lifpay-test

                    # 3. 验证部署
                    kubectl get pods -n lifpay-test -l app=breez-lnurl
                    kubectl get svc -n lifpay-test breez-lnurl-service
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