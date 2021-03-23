import matplotlib.pyplot as plt

arr = {1: 0.009, 110: 0.051, 121: 0.048, 122500: 0.171, 15: 0.024, 15400: 0.11, 18: 0.032, 1892: 0.083,
       1936: 0.085, 2: 0.009, 231: 0.057, 242: 0.057, 245000: 0.257, 25: 0.032, 30: 0.032, 30625: 0.123,
       36: 0.04, 3828: 0.091, 3872: 0.089, 4: 0.016, 462: 0.066, 484: 0.066, 490000: 0.598, 50: 0.04,
       6: 0.016, 60: 0.04, 61250: 0.141, 66: 0.048, 7656: 0.097, 7744: 0.1, 9: 0.024, 946: 0.074,
       968: 0.074}

exploreCountsByArea = {1: 24718, 110: 35, 121: 3156, 122500: 14, 15: 8506, 15400: 101, 18: 2651,
                       1892: 60, 1936: 441, 2: 18950, 231: 475, 242: 1604, 245000: 7, 25: 521, 30: 3380,
                       30625: 55, 36: 3498, 3828: 223, 3872: 76, 4: 17482, 462: 43, 484: 1253,
                       490000: 7, 50: 35, 6: 16003, 60: 548, 61250: 28, 66: 4480, 7656: 84, 7744: 87,
                       9: 15480, 946: 345, 968: 468}

exploreRequestTimeByArea = {1: 211225832151, 110: 1781439488, 121: 153013116760, 122500: 2395229140,
                            15: 206815119267, 15400: 11142794115, 18: 84337043074, 1892: 4990586977,
                            1936: 37341745125, 2: 162755452452, 231: 26924813706, 242: 90819051694,
                            245000: 1798455248, 25: 16750155386, 30: 109671089070, 30625: 6747332544,
                            36: 139556832924, 3828: 20366407985, 3872: 6779504699, 4: 282541278809,
                            462: 2819092902, 484: 82237838850, 490000: 4182882999, 50: 1382602034,
                            6: 260786664660, 60: 21716243022, 61250: 3959690727, 66: 216494016726,
                            7656: 8113779342, 7744: 8722885763, 9: 369990494082, 946: 25482063725,
                            968: 34457199870}

if __name__ == '__main__':
    # print(np.mean(arr))
    # print(np.std(arr))
    # print(np.max(arr))
    # print(np.min(arr))
    # print(np.sum(arr))
    #
    # arr = np.sort(arr)

    lists = sorted(exploreRequestTimeByArea.items())  # sorted by key, return a list of tuples

    x, y = zip(*lists)  # unpack a list of pairs into two tuples

    plt.plot(x, y)

    plt.show()
