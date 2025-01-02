package parameter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadParameters(t *testing.T) {
	env_test := "dev"
	app_test := "app_unit_test"

	_ = os.Unsetenv("PORT_APP")
	_ = os.Unsetenv("PORT_APP_1")
	_ = os.Unsetenv("PORT_APP_2")
	_ = os.Unsetenv("PORT_APP_3")
	_ = os.Unsetenv("PORT_APP_4")
	_ = os.Unsetenv("PORT_APP_5")
	_ = os.Unsetenv("PORT_APP_6")
	_ = os.Unsetenv("PORT_APP_7")
	_ = os.Unsetenv("PORT_APP_8")
	_ = os.Unsetenv("PORT_APP_9")
	_ = os.Unsetenv("PORT_APP_10")
	_ = os.Unsetenv("PORT_APP_11")
	_ = os.Unsetenv("PORT_APP_12")

	err := LoadParameters(env_test, app_test)
	assert.Nil(t, err)

	assert.NotEqual(t, "8081", os.Getenv("PORT_APP"))

	assert.Equal(t, "8080", os.Getenv("PORT_APP"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_1"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_2"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_3"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_4"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_5"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_6"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_7"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_8"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_9"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_10"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_11"))
	assert.Equal(t, "8081", os.Getenv("PORT_APP_12"))
	assert.Equal(t, "", os.Getenv("PORT_APP_13"))

}

/*
func TestLoadParameters(t *testing.T) {
	err := CreateParameter("dev", "app_test_load", "port", "8080")
	assert.Nil(t, err)

	err = CreateParameter("dev", "app_test_load", "aws_region", "us-east-2")
	assert.Nil(t, err)

	err = LoadParameters("DEV", "APP_TEST_LOAD")
	assert.Nil(t, err)
	assert.Equal(t, "8080", os.Getenv("PORT"))
	assert.Equal(t, "us-east-2", os.Getenv("AWS_REGION"))

	err = DeleteParameter("DEV", "APP_TEST_LOAD", "PORT")
	assert.Nil(t, err)

	err = DeleteParameter("DEV", "APP_TEST_LOAD", "AWS_REGION")
	assert.Nil(t, err)

	err = os.Unsetenv("PORT")
	assert.Nil(t, err)
	err = os.Unsetenv("AWS_REGION")
	assert.Nil(t, err)
}



func TestLoadParametersOverMaxResults(t *testing.T) {
	numberOfParamas := 15
	for i := 0; i < numberOfParamas; i++ {
		err := CreateParameter("dev", "test_load_many", fmt.Sprintf("load_%d", i), fmt.Sprint(i))
		assert.Nil(t, err)
	}

	err := LoadParameters("DEV", "test_load_many")
	assert.Nil(t, err)

	for i := 0; i < numberOfParamas; i++ {
		err := DeleteParameter("DEV", "test_load_many", fmt.Sprintf("load_%d", i))
		assert.Nil(t, err)
	}

	for i := 0; i < numberOfParamas; i++ {
		envVar := fmt.Sprintf("LOAD_%d", i)
		envValue := os.Getenv(envVar)
		assert.NotEmpty(t, envValue)
		err := os.Unsetenv(envVar)
		assert.Nil(t, err)
	}
}

func TestLoadParametersWithLowerCase(t *testing.T) {
	err := CreateParameter("dev", "app_test_load", "port", "8080")
	assert.Nil(t, err)

	err = CreateParameter("dev", "app_test_load", "aws_region", "us-east-2")
	assert.Nil(t, err)

	err = LoadParameters("dev", "app_test_load")
	assert.Nil(t, err)
	assert.Equal(t, "8080", os.Getenv("PORT"))
	assert.Equal(t, "us-east-2", os.Getenv("AWS_REGION"))

	err = DeleteParameter("dev", "app_test_load", "port")
	assert.Nil(t, err)

	err = DeleteParameter("dev", "app_test_load", "aws_region")
	assert.Nil(t, err)

	err = os.Unsetenv("PORT")
	assert.Nil(t, err)
	err = os.Unsetenv("AWS_REGION")
	assert.Nil(t, err)
}


func TestUpdateParameter(t *testing.T) {
	err := CreateParameter("DEV", "APP_TEST_UPDATE", "PORT_UPDATE", "8080")
	assert.Nil(t, err)

	err = LoadParameters("DEV", "APP_TEST_UPDATE")
	assert.Nil(t, err)
	assert.Equal(t, os.Getenv("PORT_UPDATE"), "8080")

	err = UpdateParameter("DEV", "APP_TEST_UPDATE", "PORT_UPDATE", "9090")
	assert.Nil(t, err)
	time.Sleep(10 * time.Second)

	err = LoadParameters("DEV", "APP_TEST_UPDATE")
	assert.Nil(t, err)
	assert.Equal(t, "9090", os.Getenv("PORT_UPDATE"))

	err = DeleteParameter("DEV", "APP_TEST_UPDATE", "PORT_UPDATE")
	assert.Nil(t, err)

	err = os.Unsetenv("PORT_UPDATE")
	assert.Nil(t, err)
}

func TestUpdateParameterWithLower(t *testing.T) {
	err := CreateParameter("dev", "app_test_update2", "port_update2", "8080")
	assert.Nil(t, err)

	err = LoadParameters("dev", "app_test_update2")
	assert.Nil(t, err)
	assert.Equal(t, os.Getenv("PORT_UPDATE2"), "8080")

	err = UpdateParameter("dev", "app_test_update2", "port_update2", "9090")
	assert.Nil(t, err)
	time.Sleep(10 * time.Second)

	err = LoadParameters("dev", "app_test_update2")
	assert.Nil(t, err)
	assert.Equal(t, "9090", os.Getenv("PORT_UPDATE2"))

	err = DeleteParameter("dev", "app_test_update2", "port_update2")
	assert.Nil(t, err)

	err = os.Unsetenv("PORT_UPDATE2")
	assert.Nil(t, err)
}

func TestDeleteParameter(t *testing.T) {
	err := CreateParameter("DEV", "APP_TEST", "PORT", "8080")
	assert.Nil(t, err)

	err = DeleteParameter("DEV", "APP_TEST", "PORT")
	assert.Nil(t, err)
}

func TestDeleteParameterWithLower(t *testing.T) {
	err := CreateParameter("dev", "app_test2", "port2", "8080")
	assert.Nil(t, err)
	err = DeleteParameter("dev", "app_test2", "port2")
	assert.Nil(t, err)
}

func TestDeleteParameterWithLowerCaseValues(t *testing.T) {
	err := CreateParameter("dev", "app_test2", "port2", "8080")
	assert.Nil(t, err)
	err = DeleteParameter("DEV", "APP_TEST2", "PORT2")
	assert.Nil(t, err)
}

func TestDeleteParameterWithInvalidName(t *testing.T) {
	err := DeleteParameter("DEV", "INVALID_APP_TEST", "INVALID_PARAM_TEST")
	assert.NotNil(t, err)
}

func TestCreateParameter(t *testing.T) {
	err := CreateParameter("DEV", "APP_NEW_TEST", "NEW_PARAM_TEST", "new parameter value")
	assert.Nil(t, err)

	err = LoadParameters("DEV", "APP_NEW_TEST")
	assert.Nil(t, err)
	assert.Equal(t, os.Getenv("NEW_PARAM_TEST"), "new parameter value")

	err = DeleteParameter("DEV", "APP_NEW_TEST", "NEW_PARAM_TEST")
	assert.Nil(t, err)

	err = os.Unsetenv("NEW_PARAM_TEST")
	assert.Nil(t, err)
}

func TestCreateParameterWithLowerCase(t *testing.T) {
	err := CreateParameter("dev", "app_new_test2", "new_param_test2", "new parameter value")
	assert.Nil(t, err)

	err = LoadParameters("dev", "app_new_test2")
	assert.Nil(t, err)
	assert.Equal(t, os.Getenv("NEW_PARAM_TEST2"), "new parameter value")

	err = DeleteParameter("dev", "app_new_test2", "NEW_PARAM_TEST2")
	assert.Nil(t, err)

	err = os.Unsetenv("NEW_PARAM_TEST2")
	assert.Nil(t, err)
}

func TestCreateParameterWithWrongSession(t *testing.T) {
	err := CreateParameter("DEV", "APP_NEW_TEST", "NEW_PARAM_TEST", "new parameter value")
	assert.Nil(t, err)

	err = LoadParameters("DEV", "APP_NEW_TEST")
	assert.Nil(t, err)
	assert.Equal(t, os.Getenv("NEW_PARAM_TEST"), "new parameter value")

	err = DeleteParameter("DEV", "APP_NEW_TEST", "NEW_PARAM_TEST")
	assert.Nil(t, err)

	err = os.Unsetenv("NEW_PARAM_TEST")
	assert.Nil(t, err)
}
*/
