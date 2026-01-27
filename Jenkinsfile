pipeline {
    agent any
    environment {
        DATABASE_URL=credentials('DB_URL') \
        SERVER_EXTERNAL_URL=credentials('SERVER_URL') \
        GITHUB_CREDENTIAL_ID = "github-id"
        APP_NAME = "breez-lnurl"
        DOCKER_IMAGE = "${APP_NAME}:latest"
        CONTAINER_NAME = "${APP_NAME}-container"
        PORT_MAPPING = "3001:8080"
        BUILD_DIR = "./build"  // 专门存放编译产物的目录
    }


    stages {
        stage('拉取代码') {
            steps {
                git(url: 'https://github.com/ReaFstar/breez-lnurl.git', credentialsId: 'github-token', branch: 'main', changelog: true, poll: false)
                sh 'ls -al && cat go.mod'
            }
        }

        stage('项目编译') {
            steps {
                sh 'mkdir -p ${BUILD_DIR}'
                sh 'go mod tidy'
                sh 'CGO_ENABLED=0 GOOS=linux go build -o app ./'
                sh 'ls ${BUILD_DIR}/app'
            }
        }

        stage('构建镜像') {
            steps {
                 sh 'ls ${BUILD_DIR}/app'
                 sh 'docker build -t ${DOCKER_IMAGE} .'
                 sh 'docker images | grep ${DOCKER_IMAGE}'

            }
        }

        stage('启动Docker容器') {
            steps {
                 sh '''
                     docker run -d \
                         --name ${CONTAINER_NAME} \
                         --restart=always \
                         -p ${PORT_MAPPING} \
                         -e DATABASE_URL=${DATABASE_URL} \
                         -e SERVER_EXTERNAL_URL=${SERVER_EXTERNAL_URL} \
                         ${DOCKER_IMAGE}
                 '''

            }
        }
    }

    // 流水线结束后反馈结果
    post {
        success {
            echo "===== 流水线执行成功！Go项目已部署为Docker容器 ====="
            echo "容器名：${CONTAINER_NAME} | 镜像名：${DOCKER_IMAGE} | 访问端口：${PORT_MAPPING%:*}"
        }
        failure {
            echo "===== 流水线执行失败！====="
            // 失败时打印容器日志，方便排查
            sh "docker logs ${CONTAINER_NAME} || true"
        }
    }

}