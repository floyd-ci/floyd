cmake_minimum_required(VERSION 3.7)

if(NOT BUILD_CONFIGURATIONS OR NOT BUILD_STEPS)
  message(FATAL_ERROR "Nothing to build!")
endif()

set(config_options)
if(EXISTS "/cache.cmake")
  list(APPEND config_options "-C/cache.cmake")
endif()
if(EXISTS "/toolchain.cmake")
  list(APPEND config_options "-DCMAKE_TOOLCHAIN_FILE:FILEPATH=/toolchain.cmake")
endif()
list(APPEND config_options "-DCMAKE_INSTALL_PREFIX:PATH=/prefix")

cmake_host_system_information(RESULT NPROC QUERY NUMBER_OF_LOGICAL_CORES)

set(run_scripts)
set(pack_projects)
foreach(config IN LISTS BUILD_CONFIGURATIONS)
  set(SOURCE_DIRECTORY "/source/${SOURCE_SUBDIR}")
  set(BINARY_DIRECTORY "/binary/${config}")
  set(CONFIGURE_OPTIONS ${config_options} "-DCMAKE_BUILD_TYPE:STRING=${config}")
  configure_file("/build.cmake" "/binary/build-${config}.cmake" @ONLY)
  list(APPEND run_scripts "/binary/build-${config}.cmake")
  list(APPEND pack_projects "/binary/${config};\${CPACK_PACKAGE_NAME};ALL;/")
endforeach()

ctest_run_script(${run_scripts} RETURN_VALUE ret)
if(NOT ret EQUAL 0)
  message(FATAL_ERROR "Failed to run build script.")
endif()

if("package" IN_LIST BUILD_STEPS)
  list(GET BUILD_CONFIGURATIONS 0 first_config)
  file(WRITE "/binary/CPackConfig.cmake"
    "include(\"/binary/${first_config}/CPackConfig.cmake\")\n"
    "set(CPACK_INSTALL_CMAKE_PROJECTS \"${pack_projects}\")\n"
    )
  execute_process(COMMAND cpack --config ./CPackConfig.cmake
    WORKING_DIRECTORY "/binary"
    RESULT_VARIABLE ret
    )
  if(NOT ret EQUAL 0)
    message(FATAL_ERROR "Failed to create package.")
  endif()
endif()

set(CTEST_RUN_CURRENT_SCRIPT OFF)
